# The logging system

Any large system needs a logging system to log all events which may need to be studied for forensics, operational audit, etc. And this needs to be built on a centralised log repository.

The aim of this logging system will be to log all operations which are being performed in the system. At a lower level of verbosity, only operations which make any changes or perform any change operation on the system will be logged. At a higher level of verbosity, all web service calls will be logged, including those web service calls which effect no change in the system. Under all circumstances, there will be no logging for each SMS or each email, since the per-message logging is a separate specialised log repository with its own schema.

## The `systemlog` table

This table may be in a relational database or on some non-SQL database, depending on system decisions. However, the columns in the table will remain the same, and the database engine will need to support JSON column types.

* `when`: timestamp of when the event happened, not when the log entry was inserted. Always in UTC, never in local time.
* `who`: string, with the userID or other unique identifying token to identify the human user who is performing the operation, *e.g.* `sudhir.kannadi@ifcm.co.in` being the username of the person performing the operation, the daemon name, *e.g.* `smsxmitd` or `emaildispatchd` for internal daemons
* `remoteip`: string, IP address of remote endpoint from where the operation is being performed. Will contain the string `LOCAL` in case the operation is not being driven by a request from any remote system, *e.g.* an internal `cron` job
* `client`: integer, giving the client ID of the client which this log entry relates to. Clients will be given access to the `log_fetch()` web service call to pull out logs from the system log
* `svr`: the server name of the system (the physical computer or VM) on which the software was running which generated this log entry. Typically, in today's context, this will be some server offering web services or running a daemon. Note: this column will record the *server name*, not the *program name*. Those server names will be used here.
* `app`: a string identifying the application from where the log entry originated. For instance, each daemon will have a unique application name, e.g. `smsxmitd` or `msglogprocessor`, *etc*. Also, each collection of web service calls will have its own application name, *e.g.* `ws_sms`, `ws_mail`, `ws_admin`, *etc*
* `thread`: string identifying the thread of execution. One thread of execution will make one sequential series of log entries, we are going to be able to identify the sequences of log entries written by one thread in one application. For a simple command-line Java program which is purely sequential, this field could be the Unix process ID. For multi-threaded Java code (typical for web service call frameworks), the Java thread ID may be logged. For concurrent Go code, the Goroutine ID may be logged (is there such a thing?)
* `module`: a string identifying a module or subsystem within the application, *e.g.* `user` for the user management module, `client` for client management, `mailbatch` for email batch management calls, *etc*
* `func`: string, giving the name of the function or method which is logging the entry. For web service calls, it will be the web service call name. But within the web service call, there could be calls to other major methods or functions, in which case that method or function name must be logged here. `func` level logging must be only done with `pri` set to one of the `debug` levels. For other log entries, where there is nothing to specify as a function or method name, the value of this field must be set to `"-"`. For high priority entries, typically, there will be no `func` level information of any value, so the hyphen must be used.
* `op`: string naming the operation which was being performed, *e.g.* `newuser` for the new user creation operation. It is not enough to just log the web service call name, because some web service calls perform both create and update, for instance. The actual operation must be named.
* `onwhat`: string giving a combination of entity type and the unique ID, name, or other "primary key" information of the object instance on which the operation was being attempted, separated by a `/`, *e.g.* 
  * if the user object with username `kkmenon@filmistan.in` is being updated, then this field will carry `user/kkmenon@filmistan.in`
  * if a mail batch with batch ID 4235 is being created, then this field will carry `mailbatch/4235`
* `status`: boolean, indicating success (`true`) or failure (`false`). Status should be marked as success for informational and debug log messages where there is no success or failure.
* `message`: string, optional, containing a human readable message
* `pri`: priority or severity of the message, listed below in order from least significant and most verbose, to most critical and least verbose:
  * `debug2`: extremely verbose debugging, documenting flow of control, functions called, parameters passed
  * `debug1`: detailed debugging information with major steps in code executed
  * `debug0`: high level debugging information, detailing entry and exit into web services, database operations, *etc*. All web service call entry and exit **must be logged**, and at this priority.
  * `info`: informational messages for operations which went smoothly. All program startups and shutdowns**must be logged**, and with entries at this priority.
  * `warn`: warning messages for things which look suspicious or not quite up to the mark, but permitted successful operation
  * `err`: failure events where errors prevented operations from completing
  * `crit`: critical failure, typically like failure of `assert()` situations, data inconsistency which is normally impossible, subsystem failure or service unavailability which cannot be worked around
  * `sec`: security alert, possible security  breach
* `params`: parameters of the operation, *e.g.* the old and new values of fields in a record-edit operation, or some key additional attributes of a new user being passed to a `newuser` operation

The `params` field will be a composite block containing a set of attributes and values, whose meaning will only be interpreted in the context of the `(app, module, op)` tuple for which this log entry is being written. This composite block will therefore not have a fixed structure, and may be stored in a JSON block. It is not necessary to log each and every parameter and attribute for every operation, unless these are ***debug*** log entries. The programmer is expected to use his judgment to decide which attributes are important enough to merit logging.

For every update operation, there must be the following three fields in the `params` JSON block for every attribute is being updated: 
* `field`: a field giving the field name being updated
* `old`: a field giving the old value of the field
* `new`: a field giving the new value which replaces the old

If multiple fields are being updated, then all must be listed in the `params` block, and with a `pri` of `debug0` or `info`.

**UNDECIDED**: Other columns may be added:

* `browser`: string, optional column, indicating the browser being used to perform this operation
* `platform`: string, optional, indicating the OS being used by the human user to perform this operation
* `geo_country`: string, carrying the three-letter ISO country code of the country from which the human user is performing this operation, obtained by geo-IP mapping of `remoteip`. Will contain `LOCAL` if `remoteIP` contains `LOCAL`.
* `geo_city`: string, containing name of city as obtained by geo-IP mapping of `remoteip`. Will contain `LOCAL` if `remoteip` contains `LOCAL`.

## Logging engine

The logging engine is some piece of code, a library and/or a daemon, which takes log entries and writes them to the repository.

We will use Kafka as our logging engine. Each server will run its own local Kafka producer instance, and all Java Springboot code, daemons, *etc*, will write log entries to their local Kafka setup.

Wrapper libraries will need to be written for programmers in our various server-side programming languages (Go, Java, shellscript) to submit log entries to Kafka. These libraries will be referred to as "client libraries" in this page.

## Log filtering

The client libraries will have the intelligence to filter what they write to the Kafka endpoint. The cluster will have a list of 4-tuples of the form:
```
(svr, app, module, pri)
```
Any of the first three fields may be `*`, indicating that it'll match all entries which have any value in that field. The fourth field will specify `pri`, where a specific value of `pri` will indicate that log entries of priority equal to or exceeding the specified value of `pri` will be logged.

The actual 4-tuples will be specified in a JSON array of the form:
``` json
"logconfig": [{
    "svr": "zeus",
    "app": "sms",
    "module": "*",
    "pri": "info"
}, {
    "svr": "athena",
    "app": "*",
    "module": "*",
    "pri": "err"
}, {
    "svr": "*",
    "app": "*",
    "module": "*",
    "pri" : "crit"
}]
```
It is recommended, though not mandatory, that the last entry in the array have `*` for values of `svr`, `app`, and `module`, so that it can act as a catch-all entry and log all the important things which may not have been explicitly matched earlier.

This log filter configuration will be loaded into the cluster's Redis cache with a specific key at some point when the cluster starts up, and then every JVM or daemon will load their config from this Redis cache. The Redis key for this entry will be `"logconfig-global"` and the value will be a long JSON string containing the full JSON object `logconfig` as shown. **UNDECIDED**: we may add a mechanism to signal every daemon and JVM to reload their log config from Redis. The details of such a signalling mechanism have not yet been decided. For a JVM running web service call stack, it will almost certainly be a web service call, which must come from `localhost`, which will call the web service `log_configreload()` and will carry no parameters in the request.

The client libraries will use a global object instance (like a singleton class in Java) to hold the logging configuration data structure.

The client libraries will have a function or method called `log_insert()` which will insert logs into the Kafka endpoint, but only if the entry matches one of the configuration entries in the `logconfig` array.

## Logging client library

There will be a client library in every programming language we use for server-side programming, to handle the logging. Currently these languages include Java (Springboot) and Go.

It must be remembered that all programs running on the server will know the name of the server (the computer) and the name of the application (the program's own name). Where this information is obtained from is a separate discussion, but it is part of the global configuration framework. The logging framework will use the value of `svr` obtained from the global configuration framework.

The library will have the following methods.

### `log_init()`

This method will have one object as parameter, and this object will have various members whose values tell the function how to connect to the local Kafka endpoint. The function will return an integer value of type `boolean` to indicate whether it was successful or not. If it returns `true`, all is well. If it is `false`, the program must abort, because no server component must continue to execute if logging is not working. Before aborting, it may choose to send out some critical alert SMS messages if such functionality is built into every daemon. I think it is a very good idea to do this.

This function will fetch the log config from Redis. Therefore, global level initialisation for the program (like URIs for accessing Redis and other resources) must be complete before `log_init()` is called. This function will go into a sleep-wait cycle for at least five minutes retrying the Redis access, if it cannot get the Redis service or the required key in Redis, at the first attempt. Only after it retries for five minutes and still fails will it exit with an error.

The `log_init()` function must initialise any internal, private global variables needed to allow logging to proceed. For instance, if a file descriptor, ID, handle, or session object of some kind is needed to submit log entries to the local Kafka endpoint, then those descriptors, handles, *etc* must be initialised. Something like `private static` members or singleton objects may be initialised to hold this information.

### `log_insert()`

This call will take a log entry and submit it to the Kafka endpoint. It will take a set of input parameters which correspond to the values needed for the fields in a log entry, except for `when` and `svr`. The value for `when` will always be picked up from the system clock (in UTC) and added to the log entry, and the `svr` will be picked up from the global initialisation done by the program at startup time, either through `log_init()` or even before that.

This call will return a `boolean` return value, with `true` indicating success. It will append the log entry to a local file if it cannot write to the Kafka endpoint for any reason, and will return failure. It will also return failure if it is called before a `log_init()` or after a `log_close()`.

### `log_close()`

This call will close the endpoint descriptors or handles which are used for log submission. In some sense, this will reverse the effect of `log_init()`. If more logging needs to be done later, a fresh `log_init()` will be needed. Typically, a program will exit without bothering to call `log_close()`.

### `log_configreload()`

This call will just reload the log config from Redis, and replace the older in-memory data structure with the new one. Everything else will remain unchanged. It is not yet decided when this function will be called, but we need the function to be available.

This call will return `boolean` with `true` for success. It will give a failure if Redis could not be reached or if it detects that there was no earlier initialisation of the logging subsystem using a `log_init()`. It is an error to call `log_configreload()` after a `log_close()` or before a `log_init()`.

Note that the logging library (in each programming environment) will have a function or method called `log_configreload()`. This is separate from the web service call of the same name, which Java Springboot applications will contain. The web service call will make some basic validation checks (like whether the call is coming from `localhost`) and then call the `log_configreload()` of the logging library. They have the same name and same purpose, but operate at different levels of abstraction.

Programs like standalone daemons *etc* which do not have a web service call listener may have a signal handler for SIGUSR2 which will call `log_configreload()` together with all sorts of other config reload functions. `cron` fired scheduled jobs which start up, do their job, and quietly exit, will not make calls to `log_configreload()`. They will do their `log_init()` followed by plenty of `log_insert()`, and then exit.

## Log repository

We will use Elasticsearch to store the log entries. We will run Elasticsearch in a cluster with data sharded across multiple nodes, and a Kafka consumer will wake up every few seconds, collect all the data received on the log stream, and insert them into Elasticsearch.

## The `log_fetch()` web service call

This call will allow an authorised user to pull out log entries from the log repository.

#### Request

The request  will take a set  of parameters most of which will be optional.
``` json
{
    "from": "2023-03-20T00:00:00Z",
    "to":   "2023-03-21T00:00:00Z",
    "who":  "sam",
    "remoteip": "202.53.55",
    "client": 53,
    "svr": "aramis",
    "app": "ws_sms",
    "module": "smsbatch",
    "onwhat": "chan/235",
    "prifrom": "debug2",
    "prito": "info",
    "paramstr": "emailaddr",
    "start": 500,
    "setsize": 100
}  
```
* `from`: **mandatory**: start timestamp of the earliest log entry the caller wants
* `to`: **mandatory**: timestamp of the most recent log entry the caller wants. The time interval between `from` and `to` must be less than or equal to **`LOGS_FETCH_MAXDURATIONMINS`** minutes, which is a global config defined in the global config dictionary.
* `who`: optional, string, a substring which will match the value in the `who` field of the log entry. This will never match any entries which have `SYSTEM` in this field.
* `remoteip`: optional, string, giving a string which may match the complete or initial part of the IP address stored in this field of any entry. Will never match any entry where this field contains the string `LOCAL`. 
* `client`: optional, integer, giving the client ID of entries to match. See **Processing** below to see the matching algorithm by which the client ID is used to select entries.
* `svr`: optional, the server name of the system (the physical computer or VM) on which the software was running which generated this log entry. 
* `app`: optional, a string identifying the application from where the log entry originated. 
* `module`: optional, a string identifying a module or subsystem within the application, *e.g.* `user` for the user management module, `client` for client management, `mailbatch` for email batch management calls, *etc*.  
* `onwhat`: optional, string, a substring which will be matched with the value of this field in the log entry.
* `prifrom`, `prito`: optional, giving the priority of the log entries the caller wants, where `prifrom` specifies the lowest priority of interest and `prito` the highest priority
* `paramstr`: optional, string, will match any string in the value of any of the members in the JSON blocks of the log entries. So, this string may match the field name, the old or new values of fields, or any other values in any other field of the JSON object.
* `start`: optional, integer, the first record number of the full matching result set which the caller wants to fetch. The very first matching log entry will be numbered 1. The records will be sorted by their timestamps in their `when` fields. If omitted, the call will return entries starting with the very first matching entry.
* `setsize`: optional, integer, the number of matching records to return. If omitted, then a set size defined in global config **`LOGS_FETCH_MAXSETSIZE`** will be used.

#### Processing

This call will apply filters based on the caller's identity. The authorization algorithm will be

```
if the user is from Broadside team with OPS, INFRA or ROOT rights, then
    all matching log entries will be selected
else if the user is from a client organisation, with any rights, then
    log entries with the client ID matching this user AND
        priority levels of INFO and above
    will be selected
else if the user is from a partner organisation, with any rights, then
    log entries with client ID matching any of the clients which this partner organisation manages, or
            with client ID matching the caller's own organisation ID AND
        priority levels of INFO and above
    will be selected
end if

after this, the other filtration parameters in the request will be applied and a smaller subset selected
```
In addition, filtration based on certain fields, *e.g.* `app` or `module` will only be available for Broadside team users. And low priority log entries too will only be for Broadside team members. The result set selection algorithm will enforce that all system messages, like program startup, shutdown, system errors, *etc* will only be available for the Broadside team, since they will all have client ID `0`.

The result set will be pulled out from the Elasticsearch data store, and will always be returned sorted as per the value in `when`.

#### Response

The response will contain a JSON block with an array of log entries, with field names as given in the `systemlog` table, and their values.

Possible errors include
* `auth` if the user does not have the authority to access the data
* `invalid_data` if the value of one of the request parameters is invalid
* `nonexistent` if the given filter criteria yield a result set of zero log entries