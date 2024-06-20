import json

import cffi
from google.protobuf.message import DecodeError

from mlflow.entities import Experiment
from mlflow.exceptions import RestException
from mlflow.protos.service_pb2 import CreateExperiment, GetExperiment
from mlflow.utils.uri import resolve_uri_if_local

ffi = cffi.FFI()

lib = ffi.dlopen("libmlflow.so")

ffi.cdef(
    """
extern int64_t CreateTrackingService(char* configJSON);
extern void DestroyTrackingService(int64_t id);
extern void* TrackingServiceCreateExperiment(
    int64_t serviceID, void* requestData, int requestSize, int* responseSize);
extern void* TrackingServiceGetExperiment(
    int64_t serviceID, void* requestData, int requestSize, int* responseSize);
void free(void*);
"""
)


class GoException(RestException):
    pass


class _GoStore:
    def __init__(self, store_url, default_artifact_root):
        super().__init__(store_url, default_artifact_root)
        self.service_id = lib.CreateTrackingService(
            json.dumps(
                {
                    "DefaultArtifactRoot": resolve_uri_if_local(default_artifact_root),
                    "LogLevel": "info",
                    "StoreUrl": store_url,
                }
            ).encode("utf-8")
        )

    def __del__(self):
        lib.DestroyTrackingService(self.service_id)

    def get_experiment(self, experiment_id):
        request = GetExperiment(experiment_id=str(experiment_id))
        request_data = request.SerializeToString()
        response_size = ffi.new("int*")

        response_data = lib.TrackingServiceGetExperiment(
            self.service_id,
            request_data,
            len(request_data),
            response_size,
        )

        response_bytes = ffi.buffer(response_data, response_size[0])[:]
        lib.free(response_data)

        try:
            response = GetExperiment.Response()
            response.ParseFromString(response_bytes)
            return Experiment.from_proto(response.experiment)
        except DecodeError:
            try:
                raise GoException(json.loads(response_bytes)) from None
            except json.JSONDecodeError as e:
                raise GoException(
                    message=f"Failed to parse response: {e}",
                )

    def create_experiment(self, name, artifact_location=None, tags=None):
        request = CreateExperiment(
            name=name,
            artifact_location=artifact_location,
            tags=[tag.to_proto() for tag in tags] if tags else [],
        )
        request_data = request.SerializeToString()
        response_size = ffi.new("int*")

        response_data = lib.TrackingServiceCreateExperiment(
            self.service_id,
            request_data,
            len(request_data),
            response_size,
        )

        response_bytes = ffi.buffer(response_data, response_size[0])[:]
        lib.free(response_data)

        try:
            response = CreateExperiment.Response()
            response.ParseFromString(response_bytes)
            return response.experiment_id
        except DecodeError:
            try:
                raise GoException(json.loads(response_bytes)) from None
            except json.JSONDecodeError as e:
                raise GoException(
                    message=f"Failed to parse response: {e}",
                )


def GoStore(cls):
    return type(cls.__name__, (_GoStore, cls), {})
