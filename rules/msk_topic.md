# msk_topic

## Requirements

An MSK topic must:
- have the topic name prefixed with the team name

## Example

### Good example

Good for team `pubsub`  :
```hcl
resource "kafka_topic" "good_topic" {
	name = "pubsub.good-topic"
}
```

### Bad examples
```hcl
# topic doesn't contain the team prefix
resource "kafka_topic" "topic_whithout_prefix" {
  name = "name-without-prefix"
}
```


## Why

Each team must own their data and control the access and settings for their topics.

## How To Fix

Define the topic satisfying the [requirements](#requirements).

See [good example](#good-example)
