This is the list of web service calls which LogHarbour will support.

***FURTHER DETAILS NEEDED: APP-LEVEL AUTHZ SUPPORT, WHAT AUTHORIZATION RULES WILL APPLY, EXACT FORMAT OF DATA RETURNED, ETC. AUTHORIZATION RULES WILL NEED TO DRILL DOWN TO REALM, APP, OBJECT CLASS, AT THE LEAST. IN SOME OF THE CALLS, AN OPTIONAL USER ID FILTERING PARAMETER WILL NEED TO BE ADDED.***

# `show_activitylog()`

This call will pull out all activities performed on a specific object instance or by a specific user, for a specific number of days time window in the past starting backwards from now.

## Request

### Parameters
- **app** (required): The application identifier
- **who** (optional): The user identifier who performed the activity.
- **class** (optional): Class of the object on which the activity was performed.
- **instance_id** (optional): Instance ID of the object on which the activity was performed.
- **days** (required): The number of days to look back into the logs from the current date and time.
- **search_after_timestamp** (optional): The timestamp from the last log entry of the previous batch. This should be used along with `search_after_doc_id` to fetch the next set of logs. This will be present in all the log retrieval related web services. 
- **search_after_doc_id** (optional): The document ID from the last log entry of the previous batch. This ensures accurate pagination by specifying the exact point in the log sequence to continue fetching from. This will be present in all the log retrieval related web services. 

We'll start with the initial request and then show what a follow-up request would look like to paginate for more logs.

#### Initial Request 

This is the first request to fetch the most recent activity logs without using "search after" parameters, as we are starting fresh:

```json
{
	"app": "OnlineStore",  
	"who": "jim",  
	"class": "Order", 
	"instance_id": "456", 
	"days": 7 
}
```

- **app**: Mandatory. Specifies the application from which to retrieve logs.
- **who**: Optional. Specifies the user who performed the activities.
- **class** and **instance_id**: Optional. Specifies the class of the object and the ID of the instance on which the activity was performed.
- **days**: Mandatory. Specifies the number of days back from the current date to fetch logs.

#### Follow-up Request 

Assuming the last log entry from the initial batch had a timestamp of `"2023-02-25T14:00:00Z"` and a document ID of `"Ux19bhs..."`, the follow-up request to fetch the next set of logs would include the "search after" parameters:

```json
{
	"app": "OnlineStore", 
	"who": "jim", 
	"class": "Order", 
	"instance_id": "456",
	"days": 7, 
	"search_after_timestamp": "2023-02-25T14:00:00Z", 
	"search_after_doc_id": "Ux19bhs..."
}
```

- **search_after_timestamp** and **search_after_doc_id**: These new parameters are used to indicate the point in the log sequence from which to continue fetching logs. They are from the last log entry received in the previous request. 

#### Validations

- The **app** and **days** parameters are mandatory to ensure that the query is properly scoped.
- The **search_after_timestamp** and **search_after_doc_id** must be used together if provided. 

## Response

The response will contain the result set of all matching records. This will search both activity logs and data-change logs, and merge them into a single timeline.

```json
{
  "logs": [
    {
      "when": "...",
      "who": "jim",
      "app": "OnlineStore",
      "system": "server1",
      "module": "Orders",
      "op": "Update",
      "type": "A",
      "class": "Order",
      "instance_id": "456",
      "status": "Success",
      "error": "...",
      "remote_ip": "...",
      "msg": "Order updated successfully",
      "data": {
        
      }
    }
  ]
}
```

#### Errors
- **app_not_found**: The specified application identifier does not exist or is not recognized.
- **invalid_days**: The number of days specified is not a valid integer or is out of the acceptable range. This error will be present in all the log retrival web services.
- **pagination_error**: An error occurred during the pagination process, possibly due to incorrect search_after_timestamp or search_after_docId. This will be present in all the log retrieval related requests. 


# getUnusualIPs()

## Request

- **app** (required): The application identifier.
- **days** (required): The number of days to look back into the logs from the current date and time.

## Response

### Processing

```go
function listUnusualIPs(appID, days):
  # Aggregate logs by IP and count occurrences
  aggregatedLogs = aggregateLogsByIP(appID, days)
  
  # Calculate total number of logs to find what 1% represents
  nLogs = calculateTotalLogs(aggregatedLogs)
  onePercentThreshold = nLogs * 0.01

  # Initialize an empty list to hold unusual IPs
  unusualIPs = []

  # For each IP in the aggregated logs
  for each IP in aggregatedLogs do
      # If the count of logs for this IP is less than the 1% threshold
      if IP.count < onePercentThreshold:
          unusualIPs.append(IP.address)
  done

  return unusualIPs
```

### Response Format

```json
{
  "unusualIPs": ["203.123.4.1", "...", "..."]
}
```

### Errors


# getLogsForIP()

## Request

- **app** (required): The application identifier.
- **ip** (required): The IP address for which logs are requested.
- **days** (required): The number of days to look back into the logs from the current date and time.

## Response

```json
{
  "logs": [
    {
      "when": "...",
      "who": "...",
      "app": "...",
      "remoteIP": "203.123.4.1",
      "type": "...",
      "msg": "...",
      // rest of the fields
      "data": {}
    },
    // Additional logs for the IP
  ]
}
```

### Errors

- **invalid_ip**: The specified IP address is not valid 

# `show_highprilog()`

This call will pull out all log entries of priority higher than a certain low watermark.

## Request Parameters
- **app** (required): The application identifier
- **pri** (required): The low watermark priority level. Log entries with this priority or higher will be returned.
- **days** (required): The number of days to look back into the logs from the current date and time.

## Response 

The response will contain all log entries whose priorities are equal to or higher than the priority in the request. For example, if `Pri` is `warn`, then include `warn`, `err`, `crit`, and `sec` levels. Priority levels are described [here](https://github.com/remiges-tech/logharbour/wiki/The-logging-system#the-systemlog-table)

The response will be a JSON object containing an array of log entries that meet the specified priority criteria:

```json
{
  "status": "success",
  "logs": [
    {
      "when": "...",
      "who": "...",
      "app": "...",
      "remote_ip": "...",
      "type": "...",
      "msg": "...",
      // Other fields from the document
      "data": {
      }
    }
  ]
}
```

### Errors
- **invalid_priority**: The specified low watermark priority level is invalid or not recognized.



# `show_datachange()`

This call will show the data change history of an object by pulling out all log entries, either for an entire object or for one field of the object.

## Request 
- **app** (required): The application identifier
- **class** (optional): Class of the object on which the activity was performed.
- **instance_id** (optional): Instance ID of the object on which the activity was performed.
- **field** (optional): The specific field name within the object for which change history is requested. 
- **days** (required): The number of days to look back into the logs from the current date and time.

## Response
The response will include a timeline of log entries detailing the history of data changes for the specified object or field:


```json
{
    "logs": [
        {
            "when": "...",
            "who": "...",
            "app": "...",
            "remote_ip": "...",
            "type": "...",
            "msg": "...",
            "data": {
                "entity": "User",
                "op": "Update",
                "changes": [
                    {
                        "field": "email",
                        "old_value": "oldEmail@example.com",
                        "new_value": "newEmail@example.com"
                    }
                ]
            }
        }
    ]
}
```
#### Filtering by Field Name
When the optional `field` parameter is provided, the service filters the change history to include only those logs where the specified field was altered. 

### Errors


# `show_debuglog()`

***AUTHZ FOR THIS WILL PROBABLY NEED TO BE SOURCE IP BASED, NOT CAPABILITY BASED. WILL WE SUPPORT UNAUTHENTICATED ACCESS TO THIS WSC FROM AN AUTHORIZED IP ADDRESS?***

## Request
- **app** (required): The application identifier
- **module** (required): The module identifier within the application
- **pri** (required): The low watermark priority level. Log entries with this priority or higher will be returned.
- **trace_id** (optional): A specific trace identifier 

## Response
The response will contain debug log entries that match the request criteria:

```json
{
  "debugLogs": [
    {
      "when": "...",
      "app": "...",
      "module": "...",
      "pri": "...",
      "msg": "...",
      "trace_id": "...",
      // All the fields in the log entry
      "data": {
      }
    }
  ]
}
```

### Errors
- **invalid_priority**: The specified low watermark priority level is invalid or not recognized.
