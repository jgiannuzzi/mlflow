import json

import cffi
from google.protobuf.message import DecodeError

from mlflow.entities import Experiment, Run, ViewType
from mlflow.exceptions import MlflowException, RestException
from mlflow.protos import databricks_pb2
from mlflow.protos.service_pb2 import (
    CreateExperiment,
    CreateRun,
    DeleteExperiment,
    GetExperiment,
    GetExperimentByName,
    LogBatch,
    SearchRuns,
)
from mlflow.utils.uri import resolve_uri_if_local

ffi = cffi.FFI()

lib = ffi.dlopen("libmlflow.so")

ffi.cdef(
    """
extern int64_t CreateTrackingService(void* configData, int configSize);
extern void DestroyTrackingService(int64_t id);
extern void* TrackingServiceCreateExperiment(
    int64_t serviceID, void* requestData, int requestSize, int* responseSize);
extern void* TrackingServiceDeleteExperiment(
    int64_t serviceID, void* requestData, int requestSize, int* responseSize);
extern void* TrackingServiceGetExperiment(
    int64_t serviceID, void* requestData, int requestSize, int* responseSize);
extern void* TrackingServiceGetExperimentByName(
    int64_t serviceID, void* requestData, int requestSize, int* responseSize);
extern void* TrackingServiceCreateRun(
    int64_t serviceID, void* requestData, int requestSize, int* responseSize);
extern void* TrackingServiceSearchRuns(
    int64_t serviceID, void* requestData, int requestSize, int* responseSize);
extern void* TrackingServiceLogBatch(
    int64_t serviceID, void* requestData, int requestSize, int* responseSize);
void free(void*);
"""
)


class GoStoreException(RestException):
    pass


class _GoStore:
    def __init__(self, store_url, default_artifact_root):
        super().__init__(store_url, default_artifact_root)
        config = json.dumps(
            {
                "DefaultArtifactRoot": resolve_uri_if_local(default_artifact_root),
                "LogLevel": "info",
                "StoreUrl": store_url,
            }
        ).encode("utf-8")
        self.service_id = lib.CreateTrackingService(config, len(config))

    def __del__(self):
        lib.DestroyTrackingService(self.service_id)

    def _call_endpoint(self, endpoint, request):
        request_data = request.SerializeToString()
        response_size = ffi.new("int*")

        response_data = endpoint(
            self.service_id,
            request_data,
            len(request_data),
            response_size,
        )

        response_bytes = ffi.buffer(response_data, response_size[0])[:]
        lib.free(response_data)

        try:
            response = type(request).Response()
            response.ParseFromString(response_bytes)
            return response
        except DecodeError:
            try:
                raise GoStoreException(json.loads(response_bytes)) from None
            except json.JSONDecodeError as e:
                raise GoStoreException(
                    message=f"Failed to parse response: {e}",
                )

    def get_experiment(self, experiment_id):
        request = GetExperiment(experiment_id=str(experiment_id))
        response = self._call_endpoint(lib.TrackingServiceGetExperiment, request)
        return Experiment.from_proto(response.experiment)

    def get_experiment_by_name(self, experiment_name):
        request = GetExperimentByName(experiment_name=experiment_name)
        try:
            response = self._call_endpoint(lib.TrackingServiceGetExperimentByName, request)
            return Experiment.from_proto(response.experiment)
        except MlflowException as e:
            if e.error_code == databricks_pb2.ErrorCode.Name(
                databricks_pb2.RESOURCE_DOES_NOT_EXIST
            ):
                return None
            raise

    def create_experiment(self, name, artifact_location=None, tags=None):
        request = CreateExperiment(
            name=name,
            artifact_location=artifact_location,
            tags=[tag.to_proto() for tag in tags] if tags else [],
        )
        response = self._call_endpoint(lib.TrackingServiceCreateExperiment, request)
        return response.experiment_id

    def delete_experiment(self, experiment_id):
        request = DeleteExperiment(experiment_id=str(experiment_id))
        self._call_endpoint(lib.TrackingServiceDeleteExperiment, request)

    def create_run(self, experiment_id, user_id, start_time, tags, run_name):
        request = CreateRun(
            experiment_id=str(experiment_id),
            user_id=user_id,
            start_time=start_time,
            tags=[tag.to_proto() for tag in tags] if tags else [],
            run_name=run_name,
        )
        response = self._call_endpoint(lib.TrackingServiceCreateRun, request)
        return Run.from_proto(response.run)

    def _search_runs(
        self, experiment_ids, filter_string, run_view_type, max_results, order_by, page_token
    ):
        request = SearchRuns(
            experiment_ids=[str(experiment_id) for experiment_id in experiment_ids],
            filter=filter_string,
            run_view_type=ViewType.to_proto(run_view_type),
            max_results=max_results,
            order_by=order_by,
            page_token=page_token,
        )
        response = self._call_endpoint(lib.TrackingServiceSearchRuns, request)
        runs = [Run.from_proto(proto_run) for proto_run in response.runs]
        return runs, response.next_page_token

    def log_batch(self, run_id, metrics, params, tags):
        request = LogBatch(
            run_id=run_id,
            metrics=[metric.to_proto() for metric in metrics],
            params=[param.to_proto() for param in params],
            tags=[tag.to_proto() for tag in tags],
        )
        self._call_endpoint(lib.TrackingServiceLogBatch, request)


def GoStore(cls):
    return type(f"Go{cls.__name__}", (_GoStore, cls), {})
