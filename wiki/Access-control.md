# The authorisation model and access control

## LogHarbour is multi-tenant

LogHarbour is designed from Day 1 to run as a shared service and maintain water-tight separation between organisations or user-groups, each of whom will have full access to manage their own LogHarbour data.

## Immutability of logs

LogHarbour does not export any interface to overwrite or modify any log entries. Short of breaking into the LogHarbour service or overcoming system security, no application has any means to change or delete any log entries, once written. There is no access capability or function in the API which provides such operations.

The LogHarbour repository is stored in an ElasticSearch index. Each organisation or user-group gets their own index. Multiple indices may reside on a common set of servers. Business applications which get access to write logs into LogHarbour do not get access to overwrite, modify or delete log entries.

## Of realms, users, applications and capabilities

LogHarbour supports the idea of **realms**. In the real world, a realm may map onto an organisation or a user group. If LogHarbour is operated as a SaaS service, a realm may map onto an "account".

A realm has a log repository with the three types of logs supported by LogHarbour: activity log, data-change log and debug log. There is one-to-one mapping between a log repository and a realm.

There is no concept of a human user of LogHarbour. A business application, or a group of applications, "uses" LogHarbour, the way they use databases. Therefore, there is no concept of a user account with respect to LogHarbour.

Within a realm, an authorised application with sufficient rights can perform all operations which LogHarbour supports. All data, configuration and resources used by LogHarbour are owned by a realm. No application has rights to perform any operation across more than one realm.

Realms are defined *via* channels other than the LogHarbour API. (A command-line program will be run on a server which has direct access to the LogHarbour data store, and will add and remove realms. Therefore, realm management will be done by the same system operations team which installs new versions of LogHarbour software, takes data backups, *etc*. The teams which manage the business applications which will write logs into LogHarbour cannot manage realms.)

LogHarbour does not have its own native UI where human users log in and perform operations. LogHarbour offers a client library, which application developers may use to integrate their application with it. If a UI is felt necessary, each business application may build its own UI and integrate their server-side code with the LogHarbour client library. (The library is currently in Go, and a Java version is slated for release before 4Q2024.)

Each realm of LogHarbour comes with two access tokens: a write token and a query token. Any piece of code which links with the LogHarbour client library and has the write token to a realm can insert log entries into the LogHarbour repository. Any application which has the query token can access the repository and fetch any log entries. The two tokens are not interchangeable. If the tokens are lost, they can be regenerated, by using the same administrative tools (command-line programs) which are used to provision a new realm. But this cannot be done by the business application -- it must be done by the same system operations team which provisions new realms.

A token is an opaque bag of bytes. It is always printable ASCII, since it's a base64-encoded binary stream, and it's always less than 5 Kbytes in length.

If an intruder manages to steal a token, he may be able to write software which will access LogHarbour and carry out all operations which business applications can perform. Therefore, tokens are not to be displayed or disclosed openly.

## How a LogHarbour instance is bootstrapped

At the most fundamental level, LogHarbour is all about writing logs from one or more servers into a central LogHarbour repository -- everything else are wrappers.

Therefore, when the LogHarbour API is first used by a new organisation:
* a new realm is defined for the organisation
* a new repository is initialised in the LogHarbour data store
* Two tokens is created for this realm, and handed over to the team which manages the business applications

This is the starting point. From this point, the business application takes over, and uses the LogHarbour client library to read and write into LogHarbour.

Some command-line utilities is part of the LogHarbour product suite, which can generate and re-generate tokens for a realm. These tokens can then be embedded in the business applications which use LogHarbour, for writing log entries and querying.

## Master data for authorisation

LogHarbour has a private data store where it stores the following details about each realm:

* `id`: an automatically incremented mandatory unique integer
* `shortname`: mandatory, unique, a one-word string, always in lower-case, following the syntactic rules of identifiers in modern PL
* `longname`: mandatory, a descriptive string
* `createdat`: mandatory, timestamp
* `writetokens`, `querytokens`: one or more write tokens and query tokens associated with this realm. All the write tokens are exactly equivalent to each other, and ditto for all the query tokens. LogHarbour does not remember which specific token was used for a specific operation.
* `payload`: mandatory, JSONB, carrying all sorts of information about the LogHarbour repository, the ElasticSearch index which will be used to hold the log entries for this realm, *etc*

## The API

LogHarbour comes with client libraries in Java and Go, so that business applications may build LogHarbour into their application code. The specs are given [in this page](Client-library). It is important to study that API to understand the authorisation model and operations of LogHarbour clearly.

