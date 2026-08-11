import unittest
from datetime import datetime, timedelta, timezone

import numpy as np

from app.domain.calibration.calibrator import Calibrator
from app.domain.calibration.codec import deserialize_histogram, serialize_histogram
from app.domain.calibration.fingerprint import build_calibration_fingerprint
from app.domain.calibration.model import CalibrationResult, StoredCalibration
from app.domain.signals.signal_extractors import HistogramSignal
from app.application.detector_service import DetectorService


class _FakeImageProcessor:
    def to_hsv(self, frame):
        return frame

    def calc_histogram(self, hsv):
        histogram = np.asarray(hsv, dtype=np.float32).reshape(-1)
        norm = np.linalg.norm(histogram)
        return histogram / norm

    def compare_histograms(self, first, second):
        return float(np.corrcoef(first, second)[0, 1])


class _FakeFrameInput:
    def __init__(self):
        self.camera_url = 'rtsp://camera.local/stream'

    def update_url(self, camera_url):
        self.camera_url = camera_url


class _FakeRepository:
    def __init__(self, return_stored=False):
        self.return_stored = return_stored
        self.load_calls = []
        self.saved = None

    def load_active(self, **kwargs):
        self.load_calls.append(kwargs)
        if not self.return_stored:
            return None
        return StoredCalibration(
            calibration_id=41,
            line_id=kwargs['line_id'],
            device_id='previous-jetson',
            config_fingerprint=kwargs['config_fingerprint'],
            algorithm_version=kwargs['algorithm_version'],
            histogram=np.linspace(0.0, 1.0, 180, dtype=np.float32),
            samples_used=30,
            quality_score=0.97,
            calibrated_at=datetime.now(timezone.utc) - timedelta(days=1),
            expires_at=datetime.now(timezone.utc) + timedelta(days=89),
            metadata={},
        )

    def save(self, **kwargs):
        self.saved = kwargs
        return 42


def _make_detector(repository):
    return DetectorService(
        frame_input=_FakeFrameInput(),
        config_port=object(),
        event_output=object(),
        image_processor=_FakeImageProcessor(),
        device_id='jetson-new',
        line_id='line-3',
        calibration_repository=repository,
        calibration_required_samples=3,
        calibration_min_quality=0.90,
    )


class CalibrationCodecTests(unittest.TestCase):
    def test_histogram_round_trip_preserves_values_shape_and_dtype(self):
        histogram = np.linspace(0.0, 1.0, 180, dtype=np.float32)

        payload, dtype_name, shape = serialize_histogram(histogram)
        restored = deserialize_histogram(payload, dtype_name, shape)

        self.assertEqual(restored.dtype, np.float32)
        self.assertEqual(restored.shape, (180,))
        np.testing.assert_array_equal(restored, histogram)

    def test_histogram_shape_mismatch_is_rejected(self):
        payload, dtype_name, _ = serialize_histogram(np.ones(180, dtype=np.float32))

        with self.assertRaises(ValueError):
            deserialize_histogram(payload, dtype_name, [90, 2])


class CalibrationFingerprintTests(unittest.TestCase):
    def _fingerprint(self, *, roi=(10, 20, 300, 120, 0), url=None, camera=None):
        return build_calibration_fingerprint(
            roi=roi,
            camera_url=url or 'rtsp://operator:secret@192.168.1.20:554/stream',
            camera_config=camera or {'exposure': 15},
            signal_scale=1.0,
        )

    def test_password_change_does_not_invalidate_calibration(self):
        first = self._fingerprint(
            url='rtsp://operator:first@192.168.1.20:554/stream'
        )
        second = self._fingerprint(
            url='rtsp://operator:second@192.168.1.20:554/stream'
        )
        self.assertEqual(first, second)

    def test_roi_or_exposure_change_invalidates_calibration(self):
        original = self._fingerprint()
        changed_roi = self._fingerprint(roi=(11, 20, 300, 120, 0))
        changed_exposure = self._fingerprint(camera={'exposure': 20})

        self.assertNotEqual(original, changed_roi)
        self.assertNotEqual(original, changed_exposure)

    def test_stream_channel_changes_hash_but_rotating_token_does_not(self):
        first = self._fingerprint(
            url='rtsp://camera.local/live?channel=1&token=old'
        )
        rotated_token = self._fingerprint(
            url='rtsp://camera.local/live?token=new&channel=1'
        )
        other_channel = self._fingerprint(
            url='rtsp://camera.local/live?channel=2&token=new'
        )

        self.assertEqual(first, rotated_token)
        self.assertNotEqual(first, other_channel)


class CalibratorTests(unittest.TestCase):
    def setUp(self):
        self.signal = HistogramSignal()
        self.signal.set_image_processor(_FakeImageProcessor())

    def test_missing_reference_is_not_learned_from_first_live_frame(self):
        frame = np.linspace(1.0, 2.0, 180, dtype=np.float32)

        score = self.signal.compute(frame, frame)

        self.assertEqual(score, 0.0)
        self.assertFalse(self.signal.has_reference)

    def test_calibrator_uses_all_samples_and_returns_quality(self):
        calibrator = Calibrator(required_samples=5)
        base = np.linspace(1.0, 2.0, 180, dtype=np.float32)
        calibrator.start()
        for offset in (-0.02, -0.01, 0.0, 0.01, 0.02):
            calibrator.add_sample(base + offset)

        result = calibrator.finish(self.signal)

        self.assertIsNotNone(result)
        self.assertEqual(result.samples_used, 5)
        self.assertGreater(result.quality_score, 0.99)
        self.assertEqual(result.histogram.shape, (180,))
        self.assertFalse(calibrator.is_active)


class DetectorCalibrationLifecycleTests(unittest.TestCase):
    def test_legacy_fsm_config_maps_to_effective_textile_contract(self):
        detector = _make_detector(_FakeRepository(return_stored=False))

        detector._apply_config({
            'fsm': {
                'n_frames': 12,
                'exit_frames': 9,
                'cooldown': 30,
                'min_rearm_s': 6,
            },
        })

        self.assertEqual(detector.fsm.config.beige_confirm_frames, 12)
        self.assertEqual(detector.fsm.config.beige_exit_frames, 9)
        self.assertEqual(detector.fsm.config.rearm_beige_frames, 30)
        self.assertEqual(detector.fsm.config.min_rearm_s, 300.0)

    def test_unsafe_legacy_defaults_are_clamped_for_textile_pilot(self):
        detector = _make_detector(_FakeRepository(return_stored=False))

        detector._apply_config({
            'fsm': {
                'n_frames': 3,
                'exit_frames': 5,
                'cooldown': 8,
                'min_rearm_s': 6,
            },
        })

        self.assertEqual(detector.fsm.config.beige_confirm_frames, 10)
        self.assertEqual(detector.fsm.config.beige_exit_frames, 8)
        self.assertEqual(detector.fsm.config.rearm_beige_frames, 25)
        self.assertEqual(detector.fsm.config.min_garment_s, 30.0)
        self.assertEqual(detector.fsm.config.min_rearm_s, 300.0)

    def test_invalid_fsm_update_keeps_last_effective_config(self):
        detector = _make_detector(_FakeRepository(return_stored=False))
        previous = detector.fsm.config

        with self.assertRaises(ValueError):
            detector._apply_config({
                'fsm': {
                    'beige_low': 0.8,
                    'beige_high': 0.2,
                },
            })

        self.assertIs(detector.fsm.config, previous)

    def test_compatible_persisted_reference_is_loaded_for_same_line(self):
        repository = _FakeRepository(return_stored=True)
        detector = _make_detector(repository)

        detector._apply_config({
            'roi': [10, 20, 300, 120],
            'camera': {'url': 'rtsp://camera.local/stream', 'exposure': 15},
        })

        self.assertTrue(detector.histogram_signal.has_reference)
        self.assertEqual(detector._active_calibration_id, 41)
        self.assertEqual(detector._calibration_state, 'ready_persisted')
        self.assertEqual(repository.load_calls[0]['line_id'], 'line-3')

    def test_accepted_manual_calibration_is_persisted(self):
        repository = _FakeRepository(return_stored=False)
        detector = _make_detector(repository)
        detector._calibration_config_fingerprint = 'a' * 64
        base = np.linspace(1.0, 2.0, 180, dtype=np.float32)

        with detector._lock:
            detector._start_calibration_locked('manual')
            for offset in (-0.01, 0.0, 0.01):
                detector.calibrator.add_sample(base + offset)
            result = detector.calibrator.finish(detector.histogram_signal)
            detector._complete_calibration_locked(result)

        self.assertEqual(detector._active_calibration_id, 42)
        self.assertEqual(detector._calibration_state, 'ready_persisted')
        self.assertEqual(repository.saved['line_id'], 'line-3')
        self.assertEqual(repository.saved['samples_used'], 3)
        self.assertGreater(repository.saved['quality_score'], 0.99)

    def test_rejected_manual_calibration_keeps_previous_reference(self):
        repository = _FakeRepository(return_stored=False)
        detector = _make_detector(repository)
        previous = np.linspace(0.0, 1.0, 180, dtype=np.float32)
        detector.histogram_signal.set_reference_histogram(previous)

        with detector._lock:
            detector._start_calibration_locked('manual')
            detector._complete_calibration_locked(CalibrationResult(
                histogram=np.linspace(1.0, 0.0, 180, dtype=np.float32),
                samples_used=3,
                quality_score=0.50,
            ))

        restored = detector.histogram_signal.get_reference_histogram()
        np.testing.assert_array_equal(restored, previous)
        self.assertEqual(detector._calibration_state, 'rejected_using_previous')
        self.assertIsNone(repository.saved)


if __name__ == '__main__':
    unittest.main()
