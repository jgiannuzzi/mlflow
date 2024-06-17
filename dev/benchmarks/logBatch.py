import time

import pyperf

import mlflow


def tick():
    return int(time.time() * 1000)


client = mlflow.tracking.MlflowClient()
experiment_id = client.create_experiment(name=f"pyperf_experiment_{tick()}")
run_id = client.create_run(experiment_id=experiment_id).info.run_id


def generate_tags(tag_count):
    return [
        mlflow.entities.RunTag(key=f"tag_{i}", value=f"tag_value_{i}") for i in range(tag_count)
    ]


def generate_params(param_count):
    return [
        mlflow.entities.Param(key=f"param_{i}", value=f"param_value_{i}")
        for i in range(param_count)
    ]


def generate_metrics(metric_count):
    return [
        mlflow.entities.Metric(key=f"metric_{i}", value=i, timestamp=tick(), step=0)
        for i in range(metric_count)
    ]


def log_batch(client, run_id, tagCount, paramCount, metricCount):
    client.log_batch(
        run_id=run_id,
        tags=generate_tags(tagCount),
        synchronous=True,
        params=generate_params(paramCount),
        metrics=generate_metrics(metricCount),
    )


runner = pyperf.Runner()
runner.bench_func("log_batch", log_batch, client, run_id, 100, 10, 200)
