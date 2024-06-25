# Benchmarks

To compare the performance of the experimental Go implementation versus the current Python backend, we have set up a small K6 script. This script is meant to be run locally and gives us an initial performance impression.

Alternatively, we can test the Python API via [pyperf](https://pyperf.readthedocs.io/en/latest/index.html)

## k6

[K6](https://k6.io/) is a popular open-source load testing tool. ([docs](https://grafana.com/docs/k6/latest/))

### Current tracking server

```sh
mlflow server --backend-store-uri postgresql://postgres:postgres@localhost:5432/postgres
```

Run test via:

```sh
k6 run -e MLFLOW_TRACKING_URI=http://localhost:5000 k6LogBatchPerfScript.js -u 20 -d 30s
```

### Experimental Go flag

```sh
mlflow server --backend-store-uri postgresql://postgres:postgres@localhost:5432/postgres --experimental-go --experimental-go-opts LogLevel=error
```

Run test via:

```sh
k6 run -e MLFLOW_TRACKING_URI=http://localhost:5000 k6LogBatchPerfScript.js -u 20 -d 30s
```

### Alternative configurations

To experiment with K6, the following flags can be tweaked:

- `-u`: number of virtual users (default 1).
- `-d`: test duration limit (eg. `30s`). This is interesting to compare how many requests were made.
- `-i`: script total iteration limit (among all VUs)

## pyperf

The pyperf script is a simple python script that can be ran.
Basic usage:

    MLFLOW_TRACKING_URI=sqlite:////tmp/test.db python logBatch.py --inherit-environ MLFLOW_TRACKING_URI

You need to set the MLFLOW_TRACKING_URI and instruct pyperf to pass it through to any child processes it might spawn.

### Test Python API

Pass a database connection uri to directly run the script against the Python API.
Examples:

    MLFLOW_TRACKING_URI=sqlite:////tmp/test.db
    MLFLOW_TRACKING_URI=postgresql://postgres:postgres@localhost:5432/postgres

### Test REST API

Start the tracking server in another tab:

    mlflow server --backend-store-uri postgresql://postgres:postgres@localhost:5432/postgres -w 1

And set the `MLFLOW_TRACKING_URI` to the corresponding server endpoint:

    MLFLOW_TRACKING_URI=http://localhost:5000

To enable the experiment Go mode, append `--experimental-go --experimental-go-opts LogLevel=error` to the `mlflow server` command.
