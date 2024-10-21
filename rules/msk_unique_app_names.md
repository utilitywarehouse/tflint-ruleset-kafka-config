# `msk_unique_app_names`

## Requirements

Requires all modules using the `tls-app` in our Kafka cluster config provide a
unique name for the `cert_common_name`. This is because this name is used to
identify the ACLs for the modules.

## Example

### Bad example

``` hcl
module "my_team_example_producer" {
  source           = "../../../modules/tls-app"
  produce_topics   = [kafka_topic.some_topic.name]
  cert_common_name = "pubsub/example-app"
}

module "my_team_example_consumer" {
  source           = "../../../modules/tls-app"
  consume_topics   = [kafka_topic.some_topic.name]
  # BAD: cert_common_name is same as module above
  cert_common_name = "pubsub/example-app"
}
```

### Good example

``` hcl
module "my_team_example_producer" {
  source           = "../../../modules/tls-app"
  produce_topics   = [kafka_topic.some_topic.name]
  cert_common_name = "pubsub/example-producer"
}

module "my_team_example_consumer" {
  source           = "../../../modules/tls-app"
  consume_topics   = [kafka_topic.some_topic.name]
  # GOOD: cert_common_name is unique
  cert_common_name = "pubsub/example-consumer"
}
```
