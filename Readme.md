


## Getting Started

- Run Temporal server on Terminal 1

    ```bash
    temporal server start-dev
    ``` 
- Run temporal workflows implemented in this project on Terminal 2

    ```bash
    go test -v ./workflows --run Test_Workflow1
    ```

## Description

- `workflows/mock_api_server.go`: Implements a moc API server having the following end points
    - live_hooks
    - media_stream_to_abr_converter

- `workflows/workflow.go`: Implements a temporal workflow called `CasWorkflow`. `CasWorkflow` creates a dag of activities and 
                        executes them. Each activity is essentially a POST call to mock server to create resource instances.
                        Also implements `CleanupActivity` which is executed when workflow is cancelled or times out to cleanup all resources created by the workflow.

- [workflows/testdata/eg_workflow.yaml](workflows/testdata/eg_workflow.yaml): A sample declaration of workflow. 
    The dependency of `media_stream_to_abr_converter` on `live_hooks` is declared here as `value_expressions` in the yaml file. 
    The `values_expressions` are somewhat similar to go template variables. The values of these variables are evaluated at runtime by the workflow.

- `workflows/workflow_test.go`: Implements test cases running workflows. A temporal worker is a goroutine waiting on queue to process workflow tasks. 
                                The test cases start a temporal worker and then start a workflow. The test cases then wait for workflow to complete. 



