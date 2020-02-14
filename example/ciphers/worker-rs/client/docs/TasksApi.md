# \TasksApi

All URIs are relative to *http://localhost:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**claim_tasks**](TasksApi.md#claim_tasks) | **post** /tasques/claims | Claims a number of Tasks
[**create_task**](TasksApi.md#create_task) | **post** /tasques | Add a new Task
[**get_existing_task**](TasksApi.md#get_existing_task) | **get** /tasques/{queue}/{id} | Get a Task
[**mark_claimed_task_done**](TasksApi.md#mark_claimed_task_done) | **put** /tasques/done/{queue}/{id} | Mark Task as Done
[**mark_claimed_task_failed**](TasksApi.md#mark_claimed_task_failed) | **put** /tasques/failed/{queue}/{id} | Mark Task as Failed
[**report_on_claimed_task**](TasksApi.md#report_on_claimed_task) | **put** /tasques/reports/{queue}/{id} | Reports on a Task
[**unclaim_existing_task**](TasksApi.md#unclaim_existing_task) | **delete** /tasques/claims/{queue}/{id} | Unclaims a Task



## claim_tasks

> Vec<crate::models::TaskTask> claim_tasks(X_TASQUES_WORKER_ID, claim)
Claims a number of Tasks

Claims a number of existing Tasks.

### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**X_TASQUES_WORKER_ID** | **String** | Worker ID | [required] |
**claim** | [**TaskClaim**](TaskClaim.md) | The request body | [required] |

### Return type

[**Vec<crate::models::TaskTask>**](task.Task.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## create_task

> crate::models::TaskTask create_task(new_task)
Add a new Task

Creates a new Task

### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**new_task** | [**TaskNewTask**](TaskNewTask.md) | The request body | [required] |

### Return type

[**crate::models::TaskTask**](task.Task.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## get_existing_task

> crate::models::TaskTask get_existing_task(queue, id)
Get a Task

Retrieves a persisted Task

### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**queue** | **String** | The Queue of the Task | [required] |
**id** | **String** | The id of the Task | [required] |

### Return type

[**crate::models::TaskTask**](task.Task.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## mark_claimed_task_done

> crate::models::TaskTask mark_claimed_task_done(queue, id, X_TASQUES_WORKER_ID, success)
Mark Task as Done

Marks a claimed Task as done.

### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**queue** | **String** | The Queue of the Task | [required] |
**id** | **String** | The id of the Task | [required] |
**X_TASQUES_WORKER_ID** | **String** | Worker ID | [required] |
**success** | [**TaskSuccess**](TaskSuccess.md) | The request body | [required] |

### Return type

[**crate::models::TaskTask**](task.Task.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## mark_claimed_task_failed

> crate::models::TaskTask mark_claimed_task_failed(queue, id, X_TASQUES_WORKER_ID, failure)
Mark Task as Failed

Marks a claimed Task as failed.

### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**queue** | **String** | The Queue of the Task | [required] |
**id** | **String** | The id of the Task | [required] |
**X_TASQUES_WORKER_ID** | **String** | Worker ID | [required] |
**failure** | [**TaskFailure**](TaskFailure.md) | The request body | [required] |

### Return type

[**crate::models::TaskTask**](task.Task.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## report_on_claimed_task

> crate::models::TaskTask report_on_claimed_task(queue, id, X_TASQUES_WORKER_ID, new_report)
Reports on a Task

Reports in on a claimed Task.

### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**queue** | **String** | The Queue of the Task | [required] |
**id** | **String** | The id of the Task | [required] |
**X_TASQUES_WORKER_ID** | **String** | Worker ID | [required] |
**new_report** | [**TaskNewReport**](TaskNewReport.md) | The request body | [required] |

### Return type

[**crate::models::TaskTask**](task.Task.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## unclaim_existing_task

> crate::models::TaskTask unclaim_existing_task(queue, id, X_TASQUES_WORKER_ID)
Unclaims a Task

Unclaims a claimed Task.

### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**queue** | **String** | The Queue of the Task | [required] |
**id** | **String** | The id of the Task | [required] |
**X_TASQUES_WORKER_ID** | **String** | Worker ID | [required] |

### Return type

[**crate::models::TaskTask**](task.Task.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

