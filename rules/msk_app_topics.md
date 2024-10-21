# `msk_app_topics`

Requires that any `consume_topics` or `produce_topics` contain only topics that
are defined in the current module. This is because we want the team that defines
a topic to also control who produces and consumes from it and how.

## Example

### Bad examples

Another team defines in their module an app that consumes from a topic owned by
another team:

``` hcl
# dev-aws/kafka-shared-msk/second-team/file.tf
resource "kafka_topic" "indexer_topic" {
  name = "second-team.indexer-topic"

  replication_factor = 3
  partitions         = 10
  config = {
    # ...
  }
}

module "second_team_indexer" {
  source           = "../../../modules/tls-app"
  cert_common_name = "second-team/indexer"

  # BAD: this module doesn't define any of first-team's topics
  consume_topics = ["first-team.example"]

  # OK: produce topic is defined in this module
  produce_topics = [kafka_topic.indexer_topic.name]
  consume_groups = ["second-team.example-consumer"]
}
```

### Good example

The solution is to split up the app definition so that the consumer and
producers are defined in the modules whose topics they use

``` hcl
# dev-aws/kafka-shared-msk/first-team/file.tf
resource "kafka_topic" "example_topic" {
  name = "first-team.example"

  replication_factor = 3
  partitions         = 10
  config = {
    # ...
  }
}

# second-team defines their consumer inside first-team's module
module "second_team_indexer" {
  source           = "../../../modules/tls-app"
  cert_common_name = "second-team/example-consumer"

  # OK: consume topic is defined in this module
  consume_topics   = [kafka_topic.example_topic.name]
  consume_groups   = ["second-team.example-consumer"]
}
```

``` hcl
# dev-aws/kafka-shared-msk/second-team/file.tf
resource "kafka_topic" "indexer_topic" {
  name = "second-team.indexer-topic"

  replication_factor = 3
  partitions         = 10
  config = {
    # ...
  }
}

# second-team defines producer that users their topic in their module
module "second_team_indexer" {
  source           = "../../../modules/tls-app"
  cert_common_name = "second-team/indexer"

  # OK: produce topic is defined in this module
  produce_topics = [kafka_topic.indexer_topic.name]
}
```
