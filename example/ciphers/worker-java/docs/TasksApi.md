# TasksApi

All URIs are relative to *http://localhost:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**claimTasks**](TasksApi.md#claimTasks) | **POST** /tasques/claims | Claims a number of Tasks
[**createTask**](TasksApi.md#createTask) | **POST** /tasques | Add a new Task
[**getExistingTask**](TasksApi.md#getExistingTask) | **GET** /tasques/{queue}/{id} | Get a Task
[**markClaimedTaskDone**](TasksApi.md#markClaimedTaskDone) | **PUT** /tasques/done/{queue}/{id} | Mark Task as Done
[**markClaimedTaskFailed**](TasksApi.md#markClaimedTaskFailed) | **PUT** /tasques/failed/{queue}/{id} | Mark Task as Failed
[**reportOnClaimedTask**](TasksApi.md#reportOnClaimedTask) | **PUT** /tasques/reports/{queue}/{id} | Reports on a Task
[**unclaimExistingTask**](TasksApi.md#unclaimExistingTask) | **DELETE** /tasques/claims/{queue}/{id} | Unclaims a Task


<a name="claimTasks"></a>
# **claimTasks**
> List&lt;TaskTask&gt; claimTasks(X_TASQUES_WORKER_ID, claim)

Claims a number of Tasks

Claims a number of existing Tasks.

### Example
```java
// Import classes:
import tasques.client.ApiClient;
import tasques.client.ApiException;
import tasques.client.Configuration;
import tasques.client.models.*;
import tasques.client.api.TasksApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:8080");

    TasksApi apiInstance = new TasksApi(defaultClient);
    String X_TASQUES_WORKER_ID = "X_TASQUES_WORKER_ID_example"; // String | Worker ID
    TaskClaim claim = new TaskClaim(); // TaskClaim | The request body
    try {
      List<TaskTask> result = apiInstance.claimTasks(X_TASQUES_WORKER_ID, claim);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling TasksApi#claimTasks");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **X_TASQUES_WORKER_ID** | **String**| Worker ID |
 **claim** | [**TaskClaim**](TaskClaim.md)| The request body |

### Return type

[**List&lt;TaskTask&gt;**](TaskTask.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |

<a name="createTask"></a>
# **createTask**
> TaskTask createTask(newTask)

Add a new Task

Creates a new Task

### Example
```java
// Import classes:
import tasques.client.ApiClient;
import tasques.client.ApiException;
import tasques.client.Configuration;
import tasques.client.models.*;
import tasques.client.api.TasksApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:8080");

    TasksApi apiInstance = new TasksApi(defaultClient);
    TaskNewTask newTask = new TaskNewTask(); // TaskNewTask | The request body
    try {
      TaskTask result = apiInstance.createTask(newTask);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling TasksApi#createTask");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **newTask** | [**TaskNewTask**](TaskNewTask.md)| The request body |

### Return type

[**TaskTask**](TaskTask.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Created |  -  |
**400** | Invalid JSON |  -  |

<a name="getExistingTask"></a>
# **getExistingTask**
> TaskTask getExistingTask(queue, id)

Get a Task

Retrieves a persisted Task

### Example
```java
// Import classes:
import tasques.client.ApiClient;
import tasques.client.ApiException;
import tasques.client.Configuration;
import tasques.client.models.*;
import tasques.client.api.TasksApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:8080");

    TasksApi apiInstance = new TasksApi(defaultClient);
    String queue = "queue_example"; // String | The Queue of the Task
    String id = "id_example"; // String | The id of the Task
    try {
      TaskTask result = apiInstance.getExistingTask(queue, id);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling TasksApi#getExistingTask");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **queue** | **String**| The Queue of the Task |
 **id** | **String**| The id of the Task |

### Return type

[**TaskTask**](TaskTask.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**404** | Task does not exist |  -  |

<a name="markClaimedTaskDone"></a>
# **markClaimedTaskDone**
> TaskTask markClaimedTaskDone(queue, id, X_TASQUES_WORKER_ID, success)

Mark Task as Done

Marks a claimed Task as done.

### Example
```java
// Import classes:
import tasques.client.ApiClient;
import tasques.client.ApiException;
import tasques.client.Configuration;
import tasques.client.models.*;
import tasques.client.api.TasksApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:8080");

    TasksApi apiInstance = new TasksApi(defaultClient);
    String queue = "queue_example"; // String | The Queue of the Task
    String id = "id_example"; // String | The id of the Task
    String X_TASQUES_WORKER_ID = "X_TASQUES_WORKER_ID_example"; // String | Worker ID
    TaskSuccess success = new TaskSuccess(); // TaskSuccess | The request body
    try {
      TaskTask result = apiInstance.markClaimedTaskDone(queue, id, X_TASQUES_WORKER_ID, success);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling TasksApi#markClaimedTaskDone");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **queue** | **String**| The Queue of the Task |
 **id** | **String**| The id of the Task |
 **X_TASQUES_WORKER_ID** | **String**| Worker ID |
 **success** | [**TaskSuccess**](TaskSuccess.md)| The request body |

### Return type

[**TaskTask**](TaskTask.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | The Task is not currently claimed |  -  |
**403** | Worker currently has not claimed the Task |  -  |
**404** | Task does not exist |  -  |

<a name="markClaimedTaskFailed"></a>
# **markClaimedTaskFailed**
> TaskTask markClaimedTaskFailed(queue, id, X_TASQUES_WORKER_ID, failure)

Mark Task as Failed

Marks a claimed Task as failed.

### Example
```java
// Import classes:
import tasques.client.ApiClient;
import tasques.client.ApiException;
import tasques.client.Configuration;
import tasques.client.models.*;
import tasques.client.api.TasksApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:8080");

    TasksApi apiInstance = new TasksApi(defaultClient);
    String queue = "queue_example"; // String | The Queue of the Task
    String id = "id_example"; // String | The id of the Task
    String X_TASQUES_WORKER_ID = "X_TASQUES_WORKER_ID_example"; // String | Worker ID
    TaskFailure failure = new TaskFailure(); // TaskFailure | The request body
    try {
      TaskTask result = apiInstance.markClaimedTaskFailed(queue, id, X_TASQUES_WORKER_ID, failure);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling TasksApi#markClaimedTaskFailed");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **queue** | **String**| The Queue of the Task |
 **id** | **String**| The id of the Task |
 **X_TASQUES_WORKER_ID** | **String**| Worker ID |
 **failure** | [**TaskFailure**](TaskFailure.md)| The request body |

### Return type

[**TaskTask**](TaskTask.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | The Task is not currently claimed |  -  |
**403** | Worker currently has not claimed the Task |  -  |
**404** | Task does not exist |  -  |

<a name="reportOnClaimedTask"></a>
# **reportOnClaimedTask**
> TaskTask reportOnClaimedTask(queue, id, X_TASQUES_WORKER_ID, newReport)

Reports on a Task

Reports in on a claimed Task.

### Example
```java
// Import classes:
import tasques.client.ApiClient;
import tasques.client.ApiException;
import tasques.client.Configuration;
import tasques.client.models.*;
import tasques.client.api.TasksApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:8080");

    TasksApi apiInstance = new TasksApi(defaultClient);
    String queue = "queue_example"; // String | The Queue of the Task
    String id = "id_example"; // String | The id of the Task
    String X_TASQUES_WORKER_ID = "X_TASQUES_WORKER_ID_example"; // String | Worker ID
    TaskNewReport newReport = new TaskNewReport(); // TaskNewReport | The request body
    try {
      TaskTask result = apiInstance.reportOnClaimedTask(queue, id, X_TASQUES_WORKER_ID, newReport);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling TasksApi#reportOnClaimedTask");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **queue** | **String**| The Queue of the Task |
 **id** | **String**| The id of the Task |
 **X_TASQUES_WORKER_ID** | **String**| Worker ID |
 **newReport** | [**TaskNewReport**](TaskNewReport.md)| The request body |

### Return type

[**TaskTask**](TaskTask.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | The Task is not currently claimed |  -  |
**403** | Worker currently has not claimed the Task |  -  |
**404** | Task does not exist |  -  |

<a name="unclaimExistingTask"></a>
# **unclaimExistingTask**
> TaskTask unclaimExistingTask(queue, id, X_TASQUES_WORKER_ID)

Unclaims a Task

Unclaims a claimed Task.

### Example
```java
// Import classes:
import tasques.client.ApiClient;
import tasques.client.ApiException;
import tasques.client.Configuration;
import tasques.client.models.*;
import tasques.client.api.TasksApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:8080");

    TasksApi apiInstance = new TasksApi(defaultClient);
    String queue = "queue_example"; // String | The Queue of the Task
    String id = "id_example"; // String | The id of the Task
    String X_TASQUES_WORKER_ID = "X_TASQUES_WORKER_ID_example"; // String | Worker ID
    try {
      TaskTask result = apiInstance.unclaimExistingTask(queue, id, X_TASQUES_WORKER_ID);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling TasksApi#unclaimExistingTask");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **queue** | **String**| The Queue of the Task |
 **id** | **String**| The id of the Task |
 **X_TASQUES_WORKER_ID** | **String**| Worker ID |

### Return type

[**TaskTask**](TaskTask.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | The Task is not currently claimed |  -  |
**403** | Worker currently has not claimed the Task |  -  |
**404** | Task does not exist |  -  |

