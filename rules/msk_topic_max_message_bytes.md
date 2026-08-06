# msk_topic_max_message_bytes

## Requirements

The 'max.message.bytes' config is not required on an MSK topic, but when specified, it must be
less than or equal to 3MB (3145728 bytes).

## Example

### Good example

```hcl
# max.message.bytes not specified
resource "kafka_topic" "good_topic" {
  name               = "pubsub.good-topic"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
    "retention.ms"     = "86400000"
  }
}

# max.message.bytes within the limit
resource "kafka_topic" "good_topic_with_max_message_bytes" {
  name               = "pubsub.good-topic-with-max-message-bytes"
  replication_factor = 3
  config = {
    "cleanup.policy"    = "delete"
    "compression.type"  = "zstd"
    "retention.ms"      = "86400000"
    "max.message.bytes" = "3145728"
  }
}
```

### Bad example

```hcl
# max.message.bytes bigger than 3MB
resource "kafka_topic" "topic_with_large_max_message_bytes" {
  name               = "topic_with_large_max_message_bytes"
  replication_factor = 3
  config = {
    "cleanup.policy"    = "delete"
    "compression.type"  = "zstd"
    "retention.ms"      = "86400000"
    "max.message.bytes" = "3145729"
  }
}
```

## How To Fix

Set 'max.message.bytes' to a value less than or equal to 3145728 bytes, or remove it to use the broker default.
