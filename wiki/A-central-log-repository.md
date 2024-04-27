# A central log repository

This page describes the high level design notes for a centralised log repository.

## The philosophy

We list here the fundamental axioms or philosophical beliefs behind this design.

* A log repository is an information asset of the organisation. It does not belong to an application or a department.
* If logs cannot be queried with meaningful `where` clauses (*i.e.* query parameters), it is of very little use. A log in a flat file or a physical register is useless.
* If the log repository is not centralised, then events cannot be correlated. Correlation of disparate entries on a common time axis is one of the most critical functions of logging.
* The log repository must be append-only for programmers and applications. In SQL terms, programmers must have only `insert` access.
* A log repository is a major asset for compliance, audit, or RCA of any kind, and must be designed to facilitate this role. It is not a passive dumping ground for data.
* A logging system must be designed and implemented such that *(i)* its data can be replicated in near real-time for data protection purposes, and *(ii)* the logging never slows down the applications which generate the log entries.

## Log structure

Each log entry must have a set of fields with pre-decided semantics to facilitate querying. A schema is proposed here:

* **when**: timestamp of when the event happened, not when the log entry was inserted. Always in UTC, never in local time.
* **who**: string, with the userID or other unique identifying token to identify the human user who is performing the operation, *e.g.* `nmodi` being the username of the person performing the operation, `SYSTEM` for internal daemons
* **remoteip**: string, IP address of remote endpoint from where the operation is being performed. Will contain the string `LOCAL` in case the operation is not being driven by a request from any remote system, *e.g.* an internal `cron` job
* **system**: the FQDN, name, IP address or other identifier of the system (the physical computer or VM) on which the software was running which generated this log entry. Typically, in today's context, this will be some server displaying a web interface or offering web services, which will be triggered or used by actions performed on a browser or mobile app.
* **app**: a string identifying the application from where the log entry originated, *e.g.* `fa` for the financial accounting application
* **module**: a string identifying a module or subsystem within the application, *e.g.* `user` for the user management module
* **handle**: string form of the handle returned by `log_init()` and passed to every `log_insert()` call, see [[Logmodule#Language-independent-logging|this section below]]
* **op**: string naming the operation which was being performed, *e.g.* `newuser` for the new user creation operation
* **whatClass**: string giving the unique ID, name of the object instance on which the operation was being attempted, 
* **whatInstanceId**: string giving the unique ID, name, or other "primary key" information of the object instance on which the operation was being attempted, *e.g.* `kkmenon` being the username of the user whose user record creation was being attempted by the operation, or `201718/5923` being the invoice ID of the invoice being edited during an invoice-editing operation
* **status**: integer, 0 or 1, indicating success (1) or failure (0), or some other binary representation. Status should be marked as success for informational and debug log messages.
* **message**: string, optional, containing a human readable message
* **level**: priority or severity of the message, being one out of
  * ***debug2***: extremely verbose debugging, documenting flow of control, functions called, parameters passed
  * ***debug1***: detailed debugging information with major steps in code executed
  * ***debug0***: high level debugging information, detailing entry and exit into web services, database operations, *etc*
  * ***info***: informational messages for operations which went smoothly
  * ***warn***: warning messages for things which look suspicious or not quite up to the mark, but permitted successful operation
  * ***err***: failure events where errors prevented operations from completing
  * ***crit***: critical failure, typically like failure of `assert()` situations, data inconsistency which is normally impossible, subsystem failure or service unavailability which cannot be worked around
  * ***sec***: security alert, possible security  breach
* **params**: parameters of the operation, *e.g.* the old and new values of fields in a record-edit operation, or additional attributes of a new user being passed to a `newuser` operation
* **spanId**: spanId for open telemetry
* **correlationId**:  correlationId for open telemetry

Additional information to be printed in a debug message
* **pid**:  processId. To be printed for debug type messages *e.g.* 28761
* **runtime**:  runtime version of code writing the log. *e.g.* go1.21.1
* **source**:  file name and line no from where the log is being written *e.g.* sampleMain.go:38
* **callTrace**:  call trace i.e. call trace of the method writing the log. *e.g.* main.firstFunc
 


The **params** field will be a composite block containing a set of attributes and values, whose meaning will only be interpreted in the context of the `(app, module, op)` tuple for which this log entry is being written. This composite block will therefore not have a fixed structure, and may be stored in a JSON block. It is not necessary to log each and every parameter and attribute for every operation, unless these are ***debug*** log entries. The programmer is expected to use his judgment to decide which attributes are important enough to merit logging.

Other columns may be added as needed. The following are strongly recommended:

* **browser**: string, optional column, indicating the browser being used to perform this operation
* **platform**: string, optional, indicating the OS being used by the human user to perform this operation
* **geo_country**: string, carrying the three-letter ISO country code of the country from which the human user is performing this operation, obtained by geo-IP mapping of `remoteip`. Will contain `LOCAL` if `remoteIP` contains `LOCAL`.
* **geo_city**: string, containing name of city as obtained by geo-IP mapping of `remoteip`. Will contain `LOCAL` if `remoteip` contains `LOCAL`.

Others may also be added.

## Architecture

A log repository comprises

* one database (optionally with its passive replicas for data protection) and
* one or more logging engines.

A logging engine is some software which receives log entries from applications and then logs them asynchronously into its database. An organisation, or a data centre, needs to have one centralised log repository. This does not mean that it needs to operate only one logging engine. Multiple logging engines may be developed and deployed to operate in parallel, so that they can all insert log entries into a common database.

A logging engine may be a standalone OS-level process running on a separate server and listening over the network for log insert requests, or may be a library of functions running within the process context of an application. For instance, it's easy to design a logging engine which will be a set of Java classes, and will run in the same JVM context as a Tomcat or JBoss application server. This will allow Java code of the application to insert logs more efficiently into the central repository without incurring the overheads of a web service call -- they can call local methods in their own JVM context.

If the logging engine is designed as a standalone process, it will export web services, which can be called by any application written in any programming language. This makes such a logging engine language-agnostic, but incurs the overheads of a web service call for every log insert.

Multiple log engines running in the process contexts of applications may be a good idea if there are multiple different programming languages being used for applications. There can then be one logging engine written in Java, inside a Tomcat JVM, and another engine, written in C-sharp, running in the dotNet runtime process on Windows.

It is recommended, not mandatory, that each logging engine have two or more threads:

* one or more threads to receive log entries from calling code and append them to a shared queue of entries in RAM
* one thread to pull out log entries from the shared in-memory queue and insert them into the database by making SQL `insert` calls

This design allows the application code to insert log entries into the queue without waiting for the database inserts to finish, and also allows the database-write thread to use efficient features like multi-insert of PostgreSQL to insert multiple entries into the log database with one SQL statement.

## The repository database

The repository needs a database. Given current technology and product availability, the most attractive options are

* PostgreSQL: a separate database with a `logs` table, with a JSON block for the `params` column
* ElasticSearch or other [Lucene](https://lucene.apache.org/core/) based database, *e.g.* [Apache Solr](http://lucene.apache.org/solr/)

PostgreSQL will probably be more reliable, while ElasticSearch will support ready-made GUI for querying (*e.g.* [Kibana](https://www.elastic.co/products/kibana)) and other tools which are being developed by a more active developer community.

It is not necessary to store all log entries for the same duration. Purge programs must purge debug log entries within a few days, and other log entries may be retained as per a nuanced retention policy -- some log priorities or applications may be retained longer than others.

If the data is stored in an RDB, then a separate database must be defined for this repository, and

* a separate user account, called **`logger`** must be created for inserting log entries. Applications must be given details of only this user account and its password, and this user must have only `insert` rights on the table. This will ensure that no application can accidentally or deliberately modify or delete anything in the repository.
* A second user account, called **`logview`**, must be created with only `select` right, for browsing or querying the repository.
* A third user account, called **`logpurge`**, must be used for purging data, which must have `delete` (for purging) and `insert` rights (for logging the details of its own activity), but no `update` rights.

If a Lucene-based database is used, then these access controls will need to be implemented in some other way.

## Language independent logging calls

Since this is a centralised repository, applications written in any programming language must have the ability to insert log entries. Therefore, client libraries must be developed in various programming languages with three functions or methods:

* `log_handle = log_init()`: a call to open a database or file handle to insert logs with
* `log_write(handle, ...)`: a call which sends a log entry to the repository. A variant, called `log_write_multi()`, may also be developed to allow multiple log entries to be inserted with one call
* `log_close(handle)`: a call to close a logging handle obtained from `log_init()`. This may actually be a `noop` function internally, depending on the implementation.

The application programmers will only need to know these three calls, and will never know the database username and password of the raw database where the log entries are stored. They will never make database (*e.g.* ODBC/ JDBC) connections to the database.

These functions or methods will call web services over a network interface if the logging engine is a separate process, or will call local methods or functions in case the logging engine is implemented as a set of functions in the application's process context. The application programmer won't know the difference -- he will just compile and link with the appropriate library.

The `log_init()` call will return a handle object. This must be passed from calling function to called function, so that all the functions and modules within the application code can use the same handle for logging. In a typical web service code, the `log_init()` function should be called right at the beginning, and then the handle should be passed to all other function calls which may need it for logging things. Again, the top-level code of the web service should call `log_close()` to release resources just before exiting from the web service function. This structure implies that the log handle should be one of the parameters passed around from one function call to the next, while calling most internal functions. Repeated calls to `log_init()` should be avoided wherever possible. Besides, using one handle for one web service allows easy grouping of log entries by handle, and lets us group log entries by invocation of web service.

The logging functions of the log module will internally refer to an open database handle, because the logging module will need to write to a database. This database handle will be different from any database handles being used by the application code for its own database access. Writes to this database handle will also be outside any transaction boundaries which may be applied by the application code on its own database accesses for its business data. The logging module will typically insert logs into its log table with `autocommit on`.

## IP based access control

IP packet filters of the OS kernel must be used to allow logging only from a set of known, fixed IP addresses. This is very feasible because log entries will be generated only from servers, not directly from browsers or mobile apps.

## Replication for data protection

PostgreSQL and ElasticSearch both permit near real-time replication of data. Such replication must be configured to protect the data in the repository in case there is catastrophic failure of one database.

## Fault tolerance for high availability

For fault tolerance, it is necessary that the client libraries for the logging system probe two or more separate engines, each of which will have its own back-end database. The client library can then choose to push out logs to one or all of the logging engines.

This can be implemented fairly simply by custom code, or may be implemented using [Apache Zookeeper](https://zookeeper.apache.org/).

## Connectors for log files

Systems logs generated by operating systems and their services will not go automatically into our centralised log repository. Logs generated by ready-made applications too will not go there, because inserting logs into our repository requires that the application code make calls to `log_insert()` or `log_insert_multi()`. This means that only our custom-built applications, where we can modify the source, can be modified to log into our repository.

But we need the system logs and other logs in our repository. Therefore, we need special programs, which we are referring to as connectors, which will read log files in various formats, translate them to the format we need, add reasonable values for all the mandatory fields, and upload the data into our repository. They may do the uploading by making calls to `log_insert_multi()` or by direct access to the repository database itself.

If the repository database is ElasticSearch, then there is a powerful log-processing front-end called [Logstash](https://www.elastic.co/products/logstash), which sits upstream of ElasticSearch and translates logs into a format suitable for upload into ElasticSearch. Logstash has tools and utilities which monitor logfiles and upload their content automatically into Logstash, which then processes, translates, and uploads the data into ElasticSearch. There must be equally powerful ETL-type tools for PostgreSQL too.

## Log message design and architecture

There needs to be some sort of standardisation in the structure and content of the log messages themselves. Therefore, this section is not about the design of the *repository*, but about the *messages*. Therefore, this is a job for the *application software designer*.

Log messages become useful only if they are viewed as *information resources* which are useful to people *other than* those who designed and wrote the application code. When thinking of log messages, the following questions need to be answered by *someone*:

* Which events will trigger a log message? For instance, do you *really* want to log the fact that your code took the `then` branch or the `else` branch for every `if-then-else` statement? Even if only for debugging?
* What information will be logged in each message? What ***params*** will the message carry?
* What will be the priority (**pri** attribute) of each type of message?
* Will there be any standard message body for similar events? In other words, will the message texts be standardised?
* There will need to be a standard dictionary of values for the `app`, `module` and `op` fields, without which the same business operation will be referred to by different names in the code of different programmers, and the log data will become hard to process.
* Where will the `log_init()` and `log_close()` be called, and how will the log handle be passed down from the function which calls `log_init()` to other lower-level functions? Note that two separate threads of execution (*e.g.* two separate web services) should never use the same log handle, because sharing a log handle will make it impossible to separate out the log messages generated by different threads of execution.

And:

* Debug messages must be thought through and designed so that a developer other than the one who writes the code can use the debug messages to debug the code many moons later. This makes debug message design very important.

All of these need to be codified into a set of general rules, so that developers can understand and follow them when writing their code. This set of rules is the design and architecture of the log messages. This needs to be done irrespective of whether the current log repository is used or some other logging system is used.

## Query interface

The query interface to browse or select log entries will be application independent. Therefore, anyone with access to a generic query interface will be able to see a lot of internal information about a lot of applications, some of which may have confidentiality or privacy issues. Therefore, this interface must be protected with strict access controls.

It is preferable to develop customised interfaces to show the data needed for specific purposes, rather than building a single generic and flexible query interface. 

* It may be necessary to build some display interfaces which are like dashboards and only show some graphs, statistics and highlights. These will be used by executives who are monitoring the health of the logging system itself.
* Some interfaces may be built to show only debug entries of a specific application, and made available to developers for application debugging.
* Some interfaces may be built to show only ***crit***, ***err*** and ***warn*** entries from the logs, to allow monitoring the health of the applications. These interfaces may be used by the maintenance teams monitoring and supporting the applications.
* An interface showing only ***crit*** and ***sec*** entries may be useful for security monitoring and maintenance of all systems.

Therefore a single generic query interface may not be as practical as separate, special-purpose display interfaces.