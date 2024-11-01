# msk_topic_config_comments

## Requirements

Topic configurations expressed in milliseconds must have comments explaining the property and including the human-readable value.
The comments can be placed after the property definition on the same line or on the line before the definition.

For computing the human-readable values it considers the following:
- 1 month has 30 days
- 1 year has 365 days

It currently checks the properties:
- retention.ms: explanation must start with `keep data`
- local.retention.ms: explanation must start with `keep data in primary storage`
- max.compaction.lag.ms: explanation must start with `allow not compacted keys maximum`

## Example

### Good example

```hcl
# Good topic example with remote storage enabled
resource "kafka_topic" "good_topic" {
  name = "pubsub.good-topic"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data in primary storage for 1 day
    "local.retention.ms"    = "86400000"
    "retention.ms"          = "2592000000" # keep data for 1 month 
    "compression.type"      = "zstd"
  }
}

# Good compacted topic
resource "kafka_topic" "good topic" {
  name               = "good_topic"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "compact"
    "compression.type" = "zstd"
    # allow not compacted keys maximum for 7 days
    "max.compaction.lag.ms" = "604800000"
  }
}
```

### Bad examples

```hcl
# the value in the comment doesn't correspond to the actual value.
resource "kafka_topic" "topic_wrong_retention_comment" {
  name               = "topic_wrong_retention_comment"
  replication_factor = 3
  config = {
    # keep data for 1 day
    "retention.ms" = "172800000"
  }
}

# the value is not commented at all
resource "kafka_topic" "topic_without_retention_comment" {
  name = "topic_without_retention_comment"
  config = {
    "local.retention.ms" = "86400000"
  }
}
```

## How To Fix

The rule automatically fixes the comments. See the [requirements](#requirements)

See [good example](#good-example)
