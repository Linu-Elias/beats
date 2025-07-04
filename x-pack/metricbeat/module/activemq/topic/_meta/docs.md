This is the `topic` metricset of the ActiveMQ module.

The metricset provides metrics describing the available ActiveMQ topics, especially exchanged messages (enqueued, dequeued, expired, in-flight), connected consumers and producers.

To collect data, the module communicates with a Jolokia HTTP/REST endpoint that exposes the JMX metrics over HTTP/REST/JSON (JMX key: `org.apache.activemq:brokerName=localhost,destinationName=sample_queue,destinationType=Queue,type=Broker`).

The topic metricset comes with a predefined dashboard:

![metricbeat activemq topics overview](images/metricbeat-activemq-topics-overview.png)
