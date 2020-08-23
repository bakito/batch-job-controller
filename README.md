![Github Build](https://github.com/bakito/batch-job-controller/workflows/Github%20Build/badge.svg) [![Build Status](https://travis-ci.com/bakito/batch-job-controller.svg?branch=master)](https://travis-ci.com/bakito/batch-job-controller) [![Docker Repository on Quay](https://quay.io/repository/bakito/batch-job-controller/status "Docker Repository on Quay")](https://quay.io/repository/bakito/batch-job-controller) [![Go Report Card](https://goreportcard.com/badge/github.com/bakito/batch-job-controller)](https://goreportcard.com/report/github.com/bakito/batch-job-controller) [![GitHub Release](https://img.shields.io/github/release/bakito/batch-job-controller.svg?style=flat)](https://github.com/bakito/batch-job-controller/releases) [![Coverage Status](https://coveralls.io/repos/github/bakito/batch-job-controller/badge.svg?branch=master)](https://coveralls.io/github/bakito/batch-job-controller?branch=master)

# Batch Job Controller

The batch job controller allows executing pods on nodes of a cluster, where the number of concurrent running pods can be configured.
Each pod can report it's results back to the controller to have them exposed as metrics.

## Deployment

The controller expects the following environment variables


| Name | Value |
| --- | --- |
| NAMESPACE | The current namespace |
| CONFIG_MAP_NAME | The name of the configmap to read the config from |

## Configuration 

The configuration has to be stored in a configmap with the following values  

### config.yaml

Controller configuration

```yaml
name: ""                         # name of the controller; will also be used as prefix for the job pods
jobServiceAccount: ""            # service account to be used for the job pods. If empty the default will be used
jobNodeSelector: {}              # node selector labels to define in which nodes to run the jobs
runOnUnscheduledNodes: true    # if true, jobs are also started on nodes that are unschedulable
cronExpression: "42 3 * * *"     # the cron expression to trigger the job execution
reportHistory: 30                # number of execution reports to keep
podPoolSize: 10                  # number of concurrent job pods to run
runOnStartup: true               # if 'true' the jobs are triggered on startup of the controller
reportDirectory: "/var/www"      # directory to store and serve the reports
callbackServiceName: ""          # name of the controller service
callbackServicePort: 8090        # port of the controller callback api service
custom: {}                       # additional properties that can be used in a custom implementation
metrics:
  prefix: "foo_...."         # prefix for the metrics exposed by the controller
  gauges:                        # metric gauges that will be exposed by the jobs. The key is uses as suffix for the metrics. 
    test:                        # suffix of the metric
      help: "help ..."           # help text for the metric
      labels:                    # list of labels to be used with the metric. node and executionID are automatically added
        - label_a
        - label_b
```

### pod-template.yaml

The template of the pod to be started for each job.
When a pod is created it gets enriched by the controller specific configuration. [pkg\job\job.go](pkg\job\job.go)

## Job Pod

The job pod has the following env variables provided by the controller:

### Environment

| Name | Value |
| --- | --- |
| NAMESPACE | The current namespace |
| NODE_NAME | The name of the node it is running on |
| EXECUTION_ID | The id of the current job execution |
| CALLBACK_SERVICE_NAME | The name/host/ip of the callback service to send the report to |
| CALLBACK_SERVICE_PORT | The port of the callback service to send the report to |

### Callback

The controller exposes by default an endpoint to receive job results. The report is stored locally and metrics of the reports will be exposed.

#### URL

The report URL is by default: **${CALLBACK_SERVICE_RESULT_URL}**

#### Body

The body of the report contains the metric suffixes that are also defined in the controller config.
Each metric has a decimal value and a map where the key is the label name and value is the value to be used for the metric label.


```json
{
  "test": [
    {
      "value": 1.0,
      "labels": {
        "label_a": "AAA",
        "label_b": "BBB"
      }
    },
    {
      "value": 2.554,
      "labels": {
        "label_a": "AAA2",
        "label_b": "BBB2"
      }
    }
  ]
}
```

Example job script: [helm\batch-job-controller\bin\run.sh](helm\batch-job-controller\bin\run.sh)

### Upload additional files
Additional files can be uploaded. 

Use default **'Content-Disposition'** header or the **name** query parameter to define the name of the file. If the name is not defined an uuid is generated.
Each filename is prepended with the node name.

#### URL

The report URL is by default: **${CALLBACK_SERVICE_FILE_URL}**

### Examples

[test-queries.http](./testdata/test-queries.http)