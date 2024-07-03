import contextlib
import logging
import os
import socket
import sys
import time
from subprocess import Popen

import pytest

import mlflow
from mlflow.server import ARTIFACT_ROOT_ENV_VAR, BACKEND_STORE_URI_ENV_VAR

from tests.helper_functions import LOCALHOST, get_safe_port

_logger = logging.getLogger(__name__)


def _await_server_up_or_die(port, timeout=30):
    """Waits until the local flask server is listening on the given port."""
    _logger.info(f"Awaiting server to be up on {LOCALHOST}:{port}")
    start_time = time.time()
    while time.time() - start_time < timeout:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
            sock.settimeout(2)
            if sock.connect_ex((LOCALHOST, port)) == 0:
                _logger.info(f"Server is up on {LOCALHOST}:{port}!")
                break
        _logger.info("Server not yet up, waiting...")
        time.sleep(0.5)
    else:
        raise Exception(f"Failed to connect on {LOCALHOST}:{port} within {timeout} seconds")


@contextlib.contextmanager
def _init_server(backend_uri, root_artifact_uri, extra_env=None, app="mlflow.server:app"):
    """
    Launch a new REST server using the tracking store specified by backend_uri and root artifact
    directory specified by root_artifact_uri.
    :returns A tuple (url, process) containing the string URL of the server and a handle to the
             server process (a multiprocessing.Process object).
    """
    mlflow.set_tracking_uri(None)
    server_port = get_safe_port()

    if "MLFLOW_GO_TESTING" in os.environ:
        args = [
            sys.executable,
            "-m",
            "mlflow_go.cli",
            "server",
            "--workers",
            "1",
            "--gunicorn-opts",
            "--log-level=debug --limit-request-line=16384 --limit-request-field_size=16384",
            "--go-opts",
            "log_level=debug",
            "--backend-store-uri",
            backend_uri,
            "--default-artifact-root",
            root_artifact_uri,
            "--host",
            LOCALHOST,
            "--port",
            str(server_port),
        ]
        env = {
            **os.environ,
            **(extra_env or {}),
        }
    else:
        args = [
            sys.executable,
            "-m",
            "flask",
            "--app",
            app,
            "run",
            "--host",
            LOCALHOST,
            "--port",
            str(server_port),
        ]
        env = {
            **os.environ,
            BACKEND_STORE_URI_ENV_VAR: backend_uri,
            ARTIFACT_ROOT_ENV_VAR: root_artifact_uri,
            **(extra_env or {}),
        }

    with Popen(
        args,
        env=env,
    ) as proc:
        try:
            _await_server_up_or_die(server_port)
            url = f"http://{LOCALHOST}:{server_port}"
            _logger.info(
                f"Launching tracking server on {url} with backend URI {backend_uri} and "
                f"artifact root {root_artifact_uri}"
            )
            yield url
        finally:
            proc.terminate()


def _send_rest_tracking_post_request(tracking_server_uri, api_path, json_payload, auth=None):
    """
    Make a POST request to the specified MLflow Tracking API and retrieve the
    corresponding `requests.Response` object
    """
    import requests

    url = tracking_server_uri + api_path
    return requests.post(url, json=json_payload, auth=auth)
