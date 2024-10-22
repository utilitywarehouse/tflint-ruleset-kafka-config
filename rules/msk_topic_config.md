# msk_topic_config

## Requirements

An MSK topic configuration must comply with the following rules:
- the replication factor must be equal to 3, because we are deploying across 3 availability zones and this is the minimum we can run, since min-in-sync replicas is set to 2. 
- the 'compression.type' must always be set to `zstd`. This is a very good compression algorithm, and it is set by default for the producer in our [kafka lib](https://github.com/utilitywarehouse/uwos-go/tree/main/pubsub/kafka)
- the 'cleanup.policy' must be specified and must be one of 'delete' or 'compact'. If not specified, it is set automatically on 'delete'. See [kafka spec](https://kafka.apache.org/30/generated/topic_config.html#topicconfigs_cleanup.policy)

## Example

### Good example

```hcl
resource "kafka_topic" "good_topic" {
  name = "pubsub.good-topic"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
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

# topic with cleanup policy
resource "kafka_topic" "topic_with_wrong_cleanup_policy" {
  name = "wrong-topic"
  config = {
    "cleanup.policy" = "invalid-value"
  }
}
```

## How To Fix

Define the topic satisfying the [requirements](#requirements).

See [good example](#good-example)
