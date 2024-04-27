# Remiges LogHarbour

Remiges LogHarbour is a highly scalable product which allows business applications to write logs into a scalable full-text database *via* a distributed message stream.

## The role of LogHarbour

All business applications should maintain persistent logs. But logging at scale and speed, in the world of distributed applications, is not easy unless a leading-edge technology stack is used. Various technology and architecture choices are required to use these sophisticated components and integrate them into a reliable and high-performance system.

Remiges LogHarbour is built on the [Apache Kafka](https://kafka.apache.org/) distributed message stream system, which feeds into an [ElasticSearch](https://www.elastic.co/elasticsearch) full-text database. Kafka is a reliable and scalable platform for message streams and is the platform of choice for logging in a modern distributed environment. Kafka message streams need to be stored in a searchable database for retrieval, which is where ElasticSearch comes in.

LogHarbour takes a prescriptive approach to logging, bringing best practices to the modern distributed, sometimes cloud-first environments where modern applications are hosted. It supports three distinct types of logs, which serve three different purposes.
* **Data change logs**: these carry information about field-level changes to data, and creation and deletion of entities. This log alone is adequate to reconstruct a system-wide audit trail with all associated information about who performed each operation, from which IP address, when, *etc*. These logs are of interest to auditors, forensic analysis teams, and for business operations change tracking.
* **Activity logs**: which store an activity trace. All web service calls, all major function calls, all database operations, from a single top-level starting trigger, are stored in this log, allowing correlated browsing of all event sequences. This helps monitor activity latency and throughput, traces operations performed on the system, and enables forensic analysis.
* **Debug logs**: these are purely for programmers, both for debugging code and for understanding code for maintenance and change management purposes.

LogHarbour log entries are structured JSON, not plain-text messages. Each entry has a structure and semantics for the basic set of fields, allowing generic tools to read log entries of any application and extract meaningful insights from them. For instance, it is easy for the LogHarbour GUI to report the sequence of all operations performed by a specific human user on a specific sub-system in a specific time window, without any knowledge of the underlying business semantics. Consistent integration and use of LogHarbour in any business application, with uniform data logging conventions, dramatically increase the value of the log collection as an asset which may be mined using generic tools to generate insights.

LogHarbour provides a full system for log capture, storage, search, reporting and analytics. In addition, it brings a prescriptive approach to how logging for any large business application is best implemented. Adopting LogHarbour reduces the need to explore and integrate disparate components into a neat system, and reduces the need to define logging conventions and standards.

## The architecture of LogHarbour

LogHarbour is a product which operates as a standalone service and receives log entries from your business applications *via* its client library linkages. Its components include
* client libraries in Java and Go to allow a program to insert log entries into the three logs. These client libraries integrate Kafka producers
* a service which operates as a Kafka consumer and pulls in the log entries
* an ElasticSearch data store where all logs are stored

ElasticSearch works well in a multi-master cluster with sharded data, allowing for high horizontal scalability. We recommend a 3-way sharded data store, with each shard maintaining two replicas on two separate physical servers. This may of course be adapted to smaller or larger use-cases, and the simplest is a master-slave ElasticSearch pair, for log stores which do not expect to exceed about half a billion entries.

## Open source

Remiges LogHarbour is the intellectual property of Remiges Technologies Pvt Ltd. It is being made available to you under the terms of the [Apache Licence 2.0](https://opensource.org/license/apache-2-0/).

We build products which we use as part of the solutions we build for our clients. We are committed to maintaining them because these are our critical assets which form our technical edge. We will be happy to offer consultancy and professional services around our products, and will be thrilled to see the larger community use our products with or without our direct participation. We welcome your queries, bug reports and pull requests.

## Remiges Technologies 

[Remiges Technologies Pvt Ltd](https://remiges.tech) ("Remiges") is a private limited company incorporated in India and controlled under the Companies Act 2013 (MCA), India. Remiges is a technology-driven software projects company whose vision is to build the world's best business applications for enterprise clients, using talent, thought and technology. Remiges views themselves as a technology leader who execute projects with a product engineering mindset, and has a strong commitment to the open source community. Our clients include India's three trading exchanges (who are among the largest trading exchanges in the world in terms of transaction volumes), some of the top ten broking houses in India, both of India's securities depositories, Fortune 500 MNC manufacturing organisations, cloud-first technology startups, and others.
