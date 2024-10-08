# msk_module_backend

## Requirements
Requires an S3 backend to be defined with the following properties:
- the key as the format ${env}-${platform}/${msk-cluster}-${team-name}
- the bucket contains the environment in its name

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

// backend key doesn't follow the format and the bucket doesn't have the environment in its name
terraform {
  backend "s3" {
    bucket = "mybucket-without-env"
    key    = "key-without-team-suffix"
    region = "us-east-1"
  }
}
```

### Good example

Good for team `pubsub` in the `dev` environment, on the `AWS` platform, on the `msk-shared` cluster :
```hcl
terraform {
  backend "s3" {
    bucket = "my-dev-bucket"
    key    = "dev-aws/msk-shared-pubsub"
    region = "us-east-1"
  }
}
```

## Why

We want to avoid team mixing their states due to copy/paste issues.

With this rule, all the details for the bucket will always be specified in the kafka-cluster-config repository where we have a module per team and each team has a unique name.

Having the key and bucket name in the required format, makes sure we avoid copy/paste issues between environments, platforms, teams, etc 

## How To Fix

Define the S3 backend in the team's module, satisfying the [requirements](#requirements).

See [good example](#good-example)
