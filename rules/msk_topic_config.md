# msk_topic_config

## Requirements

An MSK topic configuration must comply with the following rules:
- the replication factor must be equal to 3, because we are deploying across 3 availability zones and this is the minimum we can run, since min-in-sync replicas is set to 2. 
- the 'compression.type' must always be set to `zstd`. This is a very good compression algorithm, and it is set by default for the producer in our [kafka lib](https://github.com/utilitywarehouse/uwos-go/tree/main/pubsub/kafka)
- the 'cleanup.policy' must be specified and must be one of 'delete' or 'compact'. If not specified, it is set automatically on 'delete'. See [kafka spec](https://kafka.apache.org/30/generated/topic_config.html#topicconfigs_cleanup.policy)

When cleanup policy is 'delete': 
- 'retention.ms' must be specified in the config map with a valid int value expressed in milliseconds
- for a retention period of 3 days or more, tiered storage must be enabled and the local.retention.ms parameter must be defined
- for a retention period less than 3 days, tiered storage must be disabled and the local.retention.ms parameter must not be defined.
  See the [AWS docs](https://docs.aws.amazon.com/msk/latest/developerguide/msk-tiered-storage.html#msk-tiered-storage-constraints).

When cleanup policy is 'compact':
- 'retention.ms' must  not be specified in the config
- tiered storage must not be enabled

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

# Good topic that doesn't require remote storage as the retention time is less than 3 days
resource "kafka_topic" "good topic" {
  name               = "good_topic"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
    "retention.ms"     = "86400000"
  }
}

# Good compacted topic
resource "kafka_topic" "good topic" {
  name               = "good_topic"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "compact"
    "compression.type" = "zstd"
  }
}
```

### Bad examples
```hcl
# topic with wrong replication factor value
resource "kafka_topic" "topic_with_wrong_replication_factor" {
  name = "wrong-topic"
  replication_factor = 6
}

# topic with wrong compression type
resource "kafka_topic" "topic_with_wrong_compression_type" {
  name = "wrong-topic"
  config = {
    "compression.type" = "gzip"
  }
}

# topic with invalid cleanup policy
resource "kafka_topic" "topic_with_wrong_cleanup_policy" {
  name = "wrong-topic"
  config = {
    "cleanup.policy" = "invalid-value"
  }
}

# no retention on topic with delete policy
resource "kafka_topic" "topic_without_retention" {
  name               = "topic_without_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
  }
}

# topic with retention time longer than 3 days requires tiered storage
resource "kafka_topic" "topic_with_more_than_3_days_retention" {
  name               = "topic_with_more_than_3_days_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "retention.ms"     = "259200001"
    "compression.type" = "zstd"
  }
}
```

## How To Fix

Define the topic satisfying the [requirements](#requirements).

See [good example](#good-example)
