# msk_topic_no_infinite_retention

## Requirements

An MSK topic SHOULD not define an infinite retention period, to not use Kafka as a database.



## Why

Infinite retention will only grow the storage, and issues may arise when replaying X years of messages.

## Alternatives
The alternative is to use a compacted topic **IF** it fits the use case.
Compacted topics keep only the last record for a partitioning key. So they can be used when:
- partitioning keys are used to identify the same entity
- the data in the messages keep the whole entity state, as in they don't contain partial updates.
  For example, having records with the whole account state information and using as partitioning keys the account id is a good use case.
  Using a compacted topic in this case makes sure that only the last state of the account is kept as a record.

Another alternative is to use a snapshotting pattern in your consumer applications to periodically persist the application's state to external storage.
This allows for finite retention in Kafka topics and enables rapid recovery by loading the last snapshot and consuming only subsequent messages.
This approach is an important part of building robust systems with event sourcing, as explained in this article on [Event Sourcing with Kafka](https://ai-academy.training/2025/05/24/event-sourcing-with-kafka-architecture-patterns-you-should-know/)



## Example

### Good example

```hcl
# Good topic example with remote storage enabled
resource "kafka_topic" "good_topic" {
  name = "pubsub.good-topic"
  replication_factor = 3
  config = {
    # keep data in hot storage for 1 day
    "local.retention.ms"    = "86400000"
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    "retention.ms"          = "2592000000"
    "compression.type"      = "zstd"
  }
}

# Infinite retention topic with ignored check until an alternative is implemented
resource "kafka_topic" "good_topic" {
  name = "pubsub.good-topic"
  replication_factor = 3
  config = {
    # keep data in hot storage for 1 day
    "local.retention.ms"    = "86400000"
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # tflint-ignore: msk_topic_no_infinite_retention # infinite retention because needed for rebuilding the transaction log
    "retention.ms"          = "-1"
    "compression.type"      = "zstd"
  }
}

```
