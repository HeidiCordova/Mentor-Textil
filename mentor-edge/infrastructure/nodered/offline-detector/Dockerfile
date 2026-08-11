ARG BASE_IMAGE=docker-vision-event-detector:latest
FROM ${BASE_IMAGE}

ARG SOURCE_MANIFEST_SHA256=unknown
ARG BUILD_UTC=unknown

LABEL org.opencontainers.image.title="Mentor textile vision detector (offline overlay)" \
      org.opencontainers.image.created="${BUILD_UTC}" \
      org.mentor.base-policy="existing-jetson-image" \
      org.mentor.source-manifest-sha256="${SOURCE_MANIFEST_SHA256}" \
      org.mentor.build-tests="cv-image-cache,event-fsm,durable-event-output,calibration-persistence"

WORKDIR /app

# The existing Jetson image supplies the NVIDIA/OpenCV and Python dependencies.
# Replace only Mentor's application and audit tests, so this build needs no
# package registry and can run with Docker's network disabled.
RUN rm -rf /app/app /app/tests
COPY app/ /app/app/
COPY tests/ /app/tests/
COPY wheels/ /tmp/wheels/

RUN test "$(uname -m)" = "aarch64" \
 && python3 -m pip install \
      --no-cache-dir \
      --no-index \
      --no-deps \
      /tmp/wheels/psycopg2_binary-2.9.9-cp310-cp310-manylinux_2_17_aarch64.manylinux2014_aarch64.whl \
 && rm -rf /tmp/wheels \
 && python3 -c "import psycopg2; print('psycopg2:', psycopg2.__version__)" \
 && python3 -m unittest discover -s tests -p "test_cv_image_processor.py" -v \
 && python3 -m unittest discover -s tests -p "test_event_fsm.py" -v \
 && python3 -m unittest discover -s tests -p "test_durable_event_output.py" -v \
 && python3 -m unittest discover -s tests -p "test_calibration_persistence.py" -v \
 && python3 -c "import app.main; print('detector import: OK')"
