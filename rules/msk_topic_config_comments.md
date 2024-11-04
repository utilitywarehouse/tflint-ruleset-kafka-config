# msk_topic_config_comments

## Requirements

Topic configurations expressed in milliseconds must have comments explaining the property and including the human-readable value.
The comments can be placed after the property definition on the same line or on the line before the definition.

For computing the human-readable values it considers the following:
- 1 month has `2629800000` millis which is 30.4375 days
- 1 year has `31556952000` millis which is 365.2425 days

Here is a table with the precomputed values:

| Number | For N months   | For N years     |  
|--------|----------------|-----------------|  
| 1      | 2,629,800,000  | 31,556,952,000  |  
| 2      | 5,259,600,000  | 63,113,904,000  |  
| 3      | 7,889,400,000  | 94,670,856,000  |  
| 4      | 10,519,200,000 | 126,227,808,000 |  
| 5      | 13,149,000,000 | 157,784,760,000 |  
| 6      | 15,778,800,000 | 189,341,712,000 |  
| 7      | 18,408,600,000 | 220,898,664,000 |  
| 8      | 21,038,400,000 | 252,455,616,000 |  
| 9      | 23,668,200,000 | 284,012,568,000 |  
| 10     | 26,298,000,000 | 315,569,520,000 |

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
