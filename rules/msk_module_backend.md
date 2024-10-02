# msk_module_backend

Requires an S3 backend to be defined, with a key that has as suffix the name of the team.

## Example

### Bad examples 
```hcl
// no s3 backend
terraform {
  
}

// backend is not S3
terraform {
  backend "local" {
  }
}

// s3 backend doesn't have details
terraform {
  backend "s3" {
  }
}

// backend key doesn't have the team's suffix
terraform {
  backend "s3" {
    bucket = "mybucket"
    key    = "key-without-team-suffix"
    region = "us-east-1"
  }
}

// Good example for team `pubsub`
terraform {
  backend "s3" {
    bucket = "mybucket"
    key    = "dev-aws/msk-pubsub"
    region = "us-east-1"
  }
}
```

### Good example

Good for team `pubsub`:
```hcl
terraform {
  backend "s3" {
    bucket = "mybucket"
    key    = "dev-aws/msk-pubsub"
    region = "us-east-1"
  }
}
```

## Why

We want to avoid team mixing their states due to copy/paste issues.
With this rule, all the details for the bucket will always be specified in the kafka-cluster-config repository where we have a module per team and each team has a unique name.

## How To Fix

Define the S3 backend in the team's module, having the key as the team's suffix.

See [good example](#good-example)
