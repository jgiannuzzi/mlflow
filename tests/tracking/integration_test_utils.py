import contextlib
import logging
import os
import socket
import sys
import time
from subprocess import Popen

import mlflow
from mlflow.server import ARTIFACT_ROOT_ENV_VAR, BACKEND_STORE_URI_ENV_VAR, _build_go_command
from mlflow.server.handlers import initialize_backend_stores

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
def _init_server(
    backend_uri, root_artifact_uri, extra_env=None, app="mlflow.server:app", go_server_path=None
):
    """
    Launch a new REST server using the tracking store specified by backend_uri and root artifact
    directory specified by root_artifact_uri.
    :returns A tuple (url, process) containing the string URL of the server and a handle to the
             server process (a multiprocessing.Process object).
    """
    mlflow.set_tracking_uri(None)
    server_port = get_safe_port()

    env = {
        **os.environ,
        BACKEND_STORE_URI_ENV_VAR: backend_uri,
        ARTIFACT_ROOT_ENV_VAR: root_artifact_uri,
        **(extra_env or {}),
    }

    def python_command_builder(host, port):
        return [
            sys.executable,
            "-m",
            "flask",
            "--app",
            app,
            "run",
            "--host",
            host,
            "--port",
            str(port),
        ]

    def go_command_builder(host, port):
        initialize_backend_stores(
            backend_store_uri=backend_uri,
            registry_store_uri=None,
            default_artifact_root=root_artifact_uri,
        )
        return _build_go_command(
            python_command_builder,
            f"LogLevel=debug,ShutdownTimeout=1s,ServerPath={go_server_path}",
            host,
            port,
            env,
        )

    command_builder = go_command_builder if go_server_path else python_command_builder
    command = command_builder(LOCALHOST, server_port)

    with Popen(command, env=env) as proc:
        _await_server_up_or_die(server_port)
        url = f"http://{LOCALHOST}:{server_port}"
        _logger.info(
            f"Launching tracking server against backend URI {backend_uri}. Server URL: {url}"
        )
        yield url
        proc.terminate()


def _send_rest_tracking_post_request(tracking_server_uri, api_path, json_payload, auth=None):
    """
    Make a POST request to the specified MLflow Tracking API and retrieve the
    corresponding `requests.Response` object
    """
    import requests

    url = tracking_server_uri + api_path
    return requests.post(url, json=json_payload, auth=auth)
