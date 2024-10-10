# msk_topic_name

## Requirements

An MSK topic must have the name prefixed with the team name or one of the configured aliases for that team.

## Configuration

```hcl
rule "msk_topic_name" {
  enabled = true
  team_aliases = {
    pubsub = ["alias_pubsub1", "alias_pubsub2"]
    iam = ["auth", "auth-customer"]
  }
}
```

`team_aliases` maps a team name to it's allowed aliases.

## Example

### Good example

Good for team `pubsub` :
```hcl
resource "kafka_topic" "good_topic1" {
	name = "pubsub.good-topic"
}

resource "kafka_topic" "good_topic2" {
	name = "alias_pubsub1.good-topic"
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
