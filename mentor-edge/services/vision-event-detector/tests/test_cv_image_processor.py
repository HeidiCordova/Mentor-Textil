import gc
import unittest
import weakref

import cv2
import numpy as np

from app.adapters.cv_image_processor import CVImageProcessor


def _solid_bgr(value: int) -> np.ndarray:
    return np.full((8, 8, 3), value, dtype=np.uint8)


def _solid_hsv(hue: int, saturation: int, value: int) -> np.ndarray:
    hsv = np.empty((8, 8, 3), dtype=np.uint8)
    hsv[:] = (hue, saturation, value)
    return cv2.cvtColor(hsv, cv2.COLOR_HSV2BGR)


class CVImageProcessorCacheTests(unittest.TestCase):
    def test_cache_keeps_strong_reference_to_source_frame(self):
        processor = CVImageProcessor()
        gray_frame = _solid_bgr(0)
        hsv_frame = _solid_bgr(255)
        gray_frame_ref = weakref.ref(gray_frame)
        hsv_frame_ref = weakref.ref(hsv_frame)

        processor.to_grayscale(gray_frame)
        processor.to_hsv(hsv_frame)
        del gray_frame
        del hsv_frame
        gc.collect()

        self.assertIsNotNone(gray_frame_ref())
        self.assertIsNotNone(hsv_frame_ref())

    def test_temporary_frames_do_not_freeze_gray_or_hsv(self):
        gray_processor = CVImageProcessor()
        hsv_processor = CVImageProcessor()

        for value in (0, 255, 0, 255):
            gray = gray_processor.to_grayscale(_solid_bgr(value))
            self.assertTrue(np.all(gray == value))

        for value in (0, 255, 0, 255):
            hsv = hsv_processor.to_hsv(_solid_bgr(value))
            self.assertEqual(int(hsv[0, 0, 2]), value)

    def test_temporary_frames_do_not_freeze_beige_ratio(self):
        processor = CVImageProcessor()
        processor.update_beige_range(
            h_min=15,
            h_max=25,
            s_min=40,
            s_max=180,
            v_min=100,
        )

        for expected, hsv_values in (
            (1.0, (20, 100, 200)),
            (0.0, (100, 220, 200)),
            (1.0, (20, 100, 200)),
            (0.0, (100, 220, 200)),
        ):
            self.assertAlmostEqual(
                processor.beige_ratio(_solid_hsv(*hsv_values)),
                expected,
            )


if __name__ == "__main__":
    unittest.main()
