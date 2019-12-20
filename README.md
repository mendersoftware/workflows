# Introduction

In order to explore possible paths for MEN-2934, we created a PoC based on:

* API service which mimics the Conductor API to start workflows

* Worker process which processes API-only tasks

* MongoDB as data storage service and message bus, using tailable cursors.

The PoC source code can be found here:
https://github.com/tranchitella/workflows

## Technical details

We developed a golang service using the [Gin Gonic](https://github.com/gin-gonic/gin) framework which mimicks the Conductor API to request the execution of workflows. The same binary can be used as worker to listen to workflows' jobs and process them.

### API

API exposes the following end-points:

* GET /status
  Healtcheck end-point

* POST /api/workflow/:name
  Start the workflow with name `name`

* GET /api/workflow/:name/:id
  Monitor the execution statos of the job with id `id` for the workflow `name`

The API uses two collections:

* `jobs`: capped collection which supports tailable cursors; we insert the workflow execution call, together with the input parameters, into this collection when the `POST /api/workflow:name` endpoint is invoked. It is not possible to remove or update documents from this collection and it will automatically recycle the documents as it reaches the capped size.

* `jobs_status`: a traditional collection used to store the job execution status. The very same document is appended to this collection as well when the `POST /api/workflow:name` endpoint is invoked with status `pending`. This document is updated by the worker to reflect the current execution status.

You start the API server with the command:

```
$ ./bin/workflows -config config.yaml server
```

The API supports automigration at startup using the `--automigrate` flag.

### Worker

The worker opens a tailable cursor on the `jobs` collections to get notified when a new workflow job is requested to run.
The job execution is started parallely in a gorouting with a concurrency limit, managed by a monitoring channel, set in the configuration file.

As the worker picks up the job from the queue it updates the `jobs_status` it updates the status to `processing` using an atomic `FindOneAndUpdate` query against the `jobs_status` collection. The document containing the `job_status` is updated after the execution of each step of the tasks. After all the tasks are performed the status is updated to `done`.
You start the worker with the command:

```
$ ./bin/workflows -config config.yaml worker
```

The worker supports automigration at startup using the `--automigrate` flag.

### Workflows

The workflows are defined as JSON files. A sample workflow follows:

```json
{
    "name": "decommission_device",
    "description": "Removes device info from all services.",
    "version": 4,
    "tasks": [
        {
            "name": "delete_device_inventory",
            "type": "http",
            "taskdef": {
            "uri": "http://www.mocky.io/v2/5de377e13000006800e9c9c2?mocky-delay=2000ms",
            "method": "PUT",
            "payload": "{\"device_id\": \"${workflow.input.device_id}\"}",
                "headers": {
                    "X-MEN-RequestID": "${workflow.input.request_id}",
                    "Authorization": "${workflow.input.authorization}"
                },
                "connectionTimeOut": 1000,
                "readTimeOut": 1000
            }
        }
    ],
    "inputParameters": [
        "device_id",
        "request_id",
        "authorization"
    ],
    "schemaVersion": 1
}
```

Workflows are loadad by the API server and the worker at start-up time from the path specified in the configuration file.

### Conclusions

The PoC is successful, the worker can execute workflows parallely, customize the HTTP calls with the parameters and manage the concurrency to not exploit the available resources.
The latency of tailable mongodb cursor is negligible.

There are known limitations that have to be worked on to reach the status of MVP:

* Support for other types of tasks (eg. running external commands)
* More verbose logging
* Task retry in case of failure
* Workflow recovery / retry in case of failure
* Improve testing (unit, acceptance, integration)
