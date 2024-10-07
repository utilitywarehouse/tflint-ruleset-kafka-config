# `msk_app_topics`

Requires that any `consume_topics` or `produce_topics` contain only topics that
are defined in the current module.

## Example

### Bad examples

``` hcl
resource "kafka_topic" "pubsub_examples" {
  name = "pubsub.examples"
}

module "consumer" {
  # bad: "some-team.some.topic.v1" not defined in this module
  consume_topics = ["some-team.some.topic.v1"]
}
```

### Good example

``` hcl
resource "kafka_topic" "pubsub_examples" {
  name = "pubsub.examples"
}


module "consumer" {
  # good: topic is in this module
  consume_topics = [kafka_topic.pubsub_examples.name]

  # good: also fine if you use the name directly
  produce_topics = "pubsub.examples"
}
```
