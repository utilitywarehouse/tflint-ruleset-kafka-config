# msk_topic_config

## Requirements

An MSK topic configuration must comply with the following rules:
- the replication factor must be equal to 3, because we are deploying across 3 availability zones and this is the minimum we can run, since min-in-sync replicas is set to 2. 

## Example

### Good example

```hcl
resource "kafka_topic" "good_topic" {
  name = "pubsub.good-topic"
  replication_factor = 3
}

```

### Bad examples
```hcl
# topic with wrong replication factor value
resource "kafka_topic" "topic_with_wrong_replication_factor" {
  name = "wrong-topic-1"
  replication_factor = 6
}
```

## How To Fix

Define the topic satisfying the [requirements](#requirements).

See [good example](#good-example)
