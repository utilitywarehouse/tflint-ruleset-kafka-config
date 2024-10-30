package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

type topicConfigTestCase struct {
	name     string
	input    string
	fixed    string
	expected helper.Issues
}

const fileName = "topics.tf"

var replicationFactorTests = []topicConfigTestCase{
	{
		name: "missing replication factor and topic name not defined",
		input: `
resource "kafka_topic" "topic_without_repl_factor_and_name" {
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "missing replication_factor: it must be equal to '3'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 2, Column: 1},
					End:      hcl.Pos{Line: 2, Column: 60},
				},
			},
		},
	},

	{
		name: "missing replication factor",
		input: `
resource "kafka_topic" "topic_without_repl_factor" {
  name = "topic_without_repl_factor"
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_without_repl_factor" {
  name               = "topic_without_repl_factor"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "missing replication_factor: it must be equal to '3'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 2, Column: 1},
					End:      hcl.Pos{Line: 2, Column: 51},
				},
			},
		},
	},
	{
		name: "incorrect replication factor",
		input: `
resource "kafka_topic" "topic_with_incorrect_repl_factor" {
  name               = "topic_with_incorrect_repl_factor"
  replication_factor = 10
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_with_incorrect_repl_factor" {
  name               = "topic_with_incorrect_repl_factor"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "the replication_factor must be equal to '3'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 4, Column: 3},
					End:      hcl.Pos{Line: 4, Column: 26},
				},
			},
		},
	},
}

var compressionTypeTests = []topicConfigTestCase{
	{
		name: "missing config attribute",
		input: `
resource "kafka_topic" "topic_without_config" {
  name = "topic_without_config"
  replication_factor = 3
}`,
		expected: []*helper.Issue{
			{
				Message: "missing config attribute: the topic configuration must be specified in a config attribute",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 2, Column: 1},
					End:      hcl.Pos{Line: 2, Column: 46},
				},
			},
		},
	},
	{
		name: "missing compression type",
		input: `
resource "kafka_topic" "topic_without_compression_type" {
  name               = "topic_without_compression_type"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_without_compression_type" {
  name               = "topic_without_compression_type"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "missing compression.type: it must be equal to 'zstd'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 9, Column: 4},
				},
			},
		},
	},
	{
		name: "wrong compression type",
		input: `
resource "kafka_topic" "topic_with_wrong_compression_type" {
  name               = "topic_with_wrong_compression_type"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "gzip"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_with_wrong_compression_type" {
  name               = "topic_with_wrong_compression_type"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "the compression.type value must be equal to 'zstd'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 7, Column: 26},
					End:      hcl.Pos{Line: 7, Column: 32},
				},
			},
		},
	},
}

var cleanupPolicyTests = []topicConfigTestCase{
	{
		name: "missing cleanup policy",
		input: `
resource "kafka_topic" "topic_without_cleanup_policy" {
  name               = "topic_without_cleanup_policy"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_without_cleanup_policy" {
  name               = "topic_without_cleanup_policy"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
    # keep data for 1 day
    "retention.ms" = "86400000"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "missing cleanup.policy: using default 'delete'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 9, Column: 4},
				},
			},
		},
	},
	{
		name: "invalid cleanup policy value",
		input: `
resource "kafka_topic" "topic_with_invalid_cleanup_policy" {
  name               = "topic_with_invalid_cleanup_policy"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "invalid-value"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "invalid cleanup.policy: it must be one of [delete, compact], but currently is 'invalid-value'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 26},
					End:      hcl.Pos{Line: 6, Column: 41},
				},
			},
		},
	},
}

var deletePolicyRetentionTimeTests = []topicConfigTestCase{
	{
		name: "no retention on topic with delete policy",
		input: `
resource "kafka_topic" "topic_without_retention" {
  name               = "topic_without_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_without_retention" {
  name               = "topic_without_retention"
  replication_factor = 3
  config = {
    "retention.ms"     = "???"
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "retention.ms must be defined on a topic with cleanup policy delete",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 8, Column: 4},
				},
			},
		},
	},
	{
		// checking that multiple fixes will be inserted correctly as the deletion policy defaults to delete and a retention template should be inserted in this case
		name: "topic without policy and without retention time",
		input: `
resource "kafka_topic" "topic_without_policy_and_retention" {
  name               = "topic_without_policy_and_retention"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_without_policy_and_retention" {
  name               = "topic_without_policy_and_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "retention.ms"     = "???"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "missing cleanup.policy: using default 'delete'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 7, Column: 4},
				},
			},
			{
				Message: "retention.ms must be defined on a topic with cleanup policy delete",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 7, Column: 4},
				},
			},
		},
	},
	{
		name: "invalid retention time",
		input: `
resource "kafka_topic" "topic_with_invalid_retention" {
  name               = "topic_with_invalid_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "retention.ms"     = "???"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "retention.ms must have a valid integer value expressed in milliseconds. Use -1 for infinite retention",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 7, Column: 26},
					End:      hcl.Pos{Line: 7, Column: 31},
				},
			},
		},
	},
}

var deletePolicyTieredStorageTests = []topicConfigTestCase{
	{
		name: "retention time of 3 days requires tiered storage",
		input: `
resource "kafka_topic" "topic_with_more_than_3_days_retention" {
  name               = "topic_with_more_than_3_days_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    # keep data for 3 days
    "retention.ms"     = "259200000"
    "compression.type" = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_with_more_than_3_days_retention" {
  name               = "topic_with_more_than_3_days_retention"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    # keep data in hot storage for 1 day
    "local.retention.ms" = "86400000"
    "cleanup.policy"     = "delete"
    # keep data for 3 days
    "retention.ms"     = "259200000"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "tiered storage must be enabled when retention time is longer than 3 days",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 10, Column: 4},
				},
			},
			{
				Message: "missing local.retention.ms when tiered storage is enabled: using default '86400000'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 10, Column: 4},
				},
			},
		},
	},
	{
		name: "infinite retention time requires tiered storage",
		input: `
resource "kafka_topic" "topic_with_infinite_retention" {
  name               = "topic_with_infinite_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    # keep data indefinitely
    "retention.ms"     = "-1"
    "compression.type" = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_with_infinite_retention" {
  name               = "topic_with_infinite_retention"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    # keep data in hot storage for 1 day
    "local.retention.ms" = "86400000"
    "cleanup.policy"     = "delete"
    # keep data indefinitely
    "retention.ms"     = "-1"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "tiered storage must be enabled when retention time is longer than 3 days",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 10, Column: 4},
				},
			},
			{
				Message: "missing local.retention.ms when tiered storage is enabled: using default '86400000'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 10, Column: 4},
				},
			},
		},
	},
	{
		name: "forgot tiered storage enabling",
		input: `
resource "kafka_topic" "topic_with_missing_tiered_storage_enabling" {
  name               = "topic_with_missing_tiered_storage_enabling"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    # keep data for 3 days
    "retention.ms" = "259200001"
    # keep data in hot storage for 1 day
    "local.retention.ms" = "86400000"
    "compression.type" = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_with_missing_tiered_storage_enabling" {
  name               = "topic_with_missing_tiered_storage_enabling"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data for 3 days
    "retention.ms" = "259200001"
    # keep data in hot storage for 1 day
    "local.retention.ms" = "86400000"
    "compression.type"   = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "tiered storage must be enabled when retention time is longer than 3 days",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 12, Column: 4},
				},
			},
		},
	},
	{
		name: "tiered storage disabled for retention period bigger than 3 days",
		input: `
resource "kafka_topic" "topic_with_more_than_3_days_retention_tiered_disabled" {
  name               = "topic_with_more_than_3_days_retention_tiered_disabled"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "false"
    "cleanup.policy"        = "delete"
    # keep data for 3 days
    "retention.ms"     = "259200001"
    "compression.type" = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_with_more_than_3_days_retention_tiered_disabled" {
  name               = "topic_with_more_than_3_days_retention_tiered_disabled"
  replication_factor = 3
  config = {
    # keep data in hot storage for 1 day
    "local.retention.ms"    = "86400000"
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data for 3 days
    "retention.ms"     = "259200001"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "tiered storage must be enabled when retention time is longer than 3 days",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 31},
					End:      hcl.Pos{Line: 6, Column: 38},
				},
			},
			{
				Message: "missing local.retention.ms when tiered storage is enabled: using default '86400000'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 11, Column: 4},
				},
			},
		},
	},
	{
		name: "tiered storage enabled without local retention",
		input: `
resource "kafka_topic" "topic_with_tiered_storage_missing_local_retention" {
  name               = "topic_with_tiered_storage_missing_local_retention"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data for 3 days
    "retention.ms"     = "259200001"
    "compression.type" = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_with_tiered_storage_missing_local_retention" {
  name               = "topic_with_tiered_storage_missing_local_retention"
  replication_factor = 3
  config = {
    # keep data in hot storage for 1 day
    "local.retention.ms"    = "86400000"
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data for 3 days
    "retention.ms"     = "259200001"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "missing local.retention.ms when tiered storage is enabled: using default '86400000'",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 3},
					End:      hcl.Pos{Line: 11, Column: 4},
				},
			},
		},
	},
	{
		name: "tiered storage enabled and local retention invalid",
		input: `
resource "kafka_topic" "topic_with_tiered_storage_local_retention_invalid" {
  name               = "topic_with_tiered_storage_local_retention_invalid"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data for 3 days
    "retention.ms"       = "259200001"
    "local.retention.ms" = "invalid-val"
    "compression.type"   = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "local.retention.ms must have a valid integer value expressed in milliseconds",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 10, Column: 28},
					End:      hcl.Pos{Line: 10, Column: 41},
				},
			},
		},
	},
	{
		name: "tiered storage enabled for less than 3 days retention",
		input: `
resource "kafka_topic" "topic_with_less_3_days_retention_with_remote_storage" {
  name               = "topic_with_less_3_days_retention_with_remote_storage"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data for 1 day
    "retention.ms"     = "86400000"
    "compression.type" = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_with_less_3_days_retention_with_remote_storage" {
  name               = "topic_with_less_3_days_retention_with_remote_storage"
  replication_factor = 3
  config = {

    "cleanup.policy" = "delete"
    # keep data for 1 day
    "retention.ms"     = "86400000"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "tiered storage is not supported for less than 3 days retention: disabling it...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 31},
					End:      hcl.Pos{Line: 6, Column: 37},
				},
			},
		},
	},
	{
		name: "tiered storage explicitly disabled for less than 3 days retention",
		input: `
resource "kafka_topic" "topic_with_less_3_days_retention_with_disabled_remote_storage" {
  name               = "topic_with_less_3_days_retention_with_disabled_remote_storage"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "false"
    "cleanup.policy"        = "delete"
    # keep data for 1 day
    "retention.ms"     = "86400000"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{},
	},
	{
		name: "local storage specified for less than 3 days retention",
		input: `
resource "kafka_topic" "topic_with_less_3_days_retention_with_local_storage" {
  name               = "topic_with_less_3_days_retention_with_local_storage"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data for 2 days
    "retention.ms"          = "172800000"
    "local.retention.ms"    = "86400000"
    "compression.type"      = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_with_less_3_days_retention_with_local_storage" {
  name               = "topic_with_less_3_days_retention_with_local_storage"
  replication_factor = 3
  config = {

    "cleanup.policy" = "delete"
    # keep data for 2 days
    "retention.ms" = "172800000"

    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "tiered storage is not supported for less than 3 days retention: disabling it...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 31},
					End:      hcl.Pos{Line: 6, Column: 37},
				},
			},
			{
				Message: "defining local.retention.ms is misleading when tiered storage is disabled due to less than 3 days retention: removing it...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 10, Column: 31},
					End:      hcl.Pos{Line: 10, Column: 41},
				},
			},
		},
	},
}

var compactPolicyTests = []topicConfigTestCase{
	{
		name: "tiered storage specified for compacted topic",
		input: `
resource "kafka_topic" "topic_compacted_with_tiered_storage" {
  name               = "topic_compacted_with_tiered_storage"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "compact"
    "compression.type"      = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_compacted_with_tiered_storage" {
  name               = "topic_compacted_with_tiered_storage"
  replication_factor = 3
  config = {

    "cleanup.policy"   = "compact"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "tiered storage is not supported for compacted topic: disabling it...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 31},
					End:      hcl.Pos{Line: 6, Column: 37},
				},
			},
		},
	},
	{
		name: "local storage specified for compacted topic",
		input: `
resource "kafka_topic" "topic_compacted_with_local_storage" {
  name               = "topic_compacted_with_local_storage"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    "local.retention.ms"    = "86400000"
    "cleanup.policy"        = "compact"
    "compression.type"      = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_compacted_with_local_storage" {
  name               = "topic_compacted_with_local_storage"
  replication_factor = 3
  config = {


    "cleanup.policy"   = "compact"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "tiered storage is not supported for compacted topic: disabling it...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 31},
					End:      hcl.Pos{Line: 6, Column: 37},
				},
			},
			{
				Message: "defining local.retention.ms is misleading when tiered storage is disabled due to compacted topic: removing it...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 7, Column: 31},
					End:      hcl.Pos{Line: 7, Column: 41},
				},
			},
		},
	},
	{
		name: "retention time specified for compacted topic",
		input: `
resource "kafka_topic" "topic_compacted_with_retention_time" {
  name               = "topic_compacted_with_retention_time"
  replication_factor = 3
  config = {
    # keep data for 1 day
    "retention.ms"     = "86400000"
    "cleanup.policy"   = "compact"
    "compression.type" = "zstd"
  }
}`,
		fixed: `
resource "kafka_topic" "topic_compacted_with_retention_time" {
  name               = "topic_compacted_with_retention_time"
  replication_factor = 3
  config = {
    # keep data for 1 day

    "cleanup.policy"   = "compact"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "defining retention.ms is misleading for compacted topic: removing it...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 7, Column: 5},
					End:      hcl.Pos{Line: 7, Column: 19},
				},
			},
		},
	},
}

var configValueCommentsTests = []topicConfigTestCase{
	{
		name: "retention time without comment",
		input: `
resource "kafka_topic" "topic_without_retention_comment" {
  name               = "topic_without_retention_comment"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "retention.ms"     = "86400000"
    "compression.type" = "zstd"
  }
}`, fixed: `
resource "kafka_topic" "topic_without_retention_comment" {
  name               = "topic_without_retention_comment"
  replication_factor = 3
  config = {
    "cleanup.policy" = "delete"
    # keep data for 1 day
    "retention.ms"     = "86400000"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "retention.ms must have a comment with the human readable value: adding it ...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 7, Column: 5},
					End:      hcl.Pos{Line: 7, Column: 19},
				},
			},
		},
	},
	{
		name: "retention time with wrong comment",
		input: `
resource "kafka_topic" "topic_wrong_retention_comment" {
  name               = "topic_wrong_retention_comment"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    # keep data for 1 day
    "retention.ms"     = "172800000"
    "compression.type" = "zstd"
  }
}`, fixed: `
resource "kafka_topic" "topic_wrong_retention_comment" {
  name               = "topic_wrong_retention_comment"
  replication_factor = 3
  config = {
    "cleanup.policy" = "delete"
    # keep data for 2 days
    "retention.ms"     = "172800000"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "retention.ms value doesn't correspond to the human readable value in the comment: fixing it ...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 7, Column: 5},
					End:      hcl.Pos{Line: 8, Column: 1},
				},
			},
		},
	},
	{
		name: "retention time good infinite comment",
		input: `
resource "kafka_topic" "topic_good_retention_comment_infinite" {
  name               = "topic_good_retention_comment_infinite"
  replication_factor = 3
  config = {
    # keep data in hot storage for 1 day
    "local.retention.ms"    = "86400000"
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data indefinitely
    "retention.ms"          = "-1"
    "compression.type"      = "zstd"
  }
}`,
		expected: []*helper.Issue{},
	},
}

var goodConfigTests = []topicConfigTestCase{
	{
		name: "good topic definition without tiered storage",
		input: `
resource "kafka_topic" "good topic" {
  name               = "good_topic"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
    # keep data for 1 day
    "retention.ms"     = "86400000"
  }
}`,
		expected: []*helper.Issue{},
	},
	{
		name: "good topic definition with tiered storage",
		input: `
resource "kafka_topic" "good topic" {
  name               = "good_topic"
  replication_factor = 3
  config = {
    # keep data in hot storage for 1 day
    "local.retention.ms"    = "86400000"
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    # keep data for 30 days
    "retention.ms"          = "2592000000"
    "compression.type"      = "zstd"
  }
}`,
		expected: []*helper.Issue{},
	},
	{
		name: "good compacted topic definition",
		input: `
resource "kafka_topic" "good topic" {
  name               = "good_topic"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "compact"
    "compression.type" = "zstd"
  }
}`,
		expected: []*helper.Issue{},
	},
}

func Test_MSKTopicConfigRule(t *testing.T) {
	rule := &MSKTopicConfigRule{}

	var allTests []topicConfigTestCase
	allTests = append(allTests, replicationFactorTests...)
	allTests = append(allTests, compressionTypeTests...)
	allTests = append(allTests, cleanupPolicyTests...)
	allTests = append(allTests, deletePolicyRetentionTimeTests...)
	allTests = append(allTests, deletePolicyTieredStorageTests...)
	allTests = append(allTests, compactPolicyTests...)
	allTests = append(allTests, configValueCommentsTests...)
	allTests = append(allTests, goodConfigTests...)

	for _, tc := range allTests {
		t.Run(tc.name, func(t *testing.T) {
			runner := helper.TestRunner(t, map[string]string{fileName: tc.input})
			require.NoError(t, rule.Check(runner))

			setExpectedRule(tc.expected, rule)
			helper.AssertIssues(t, tc.expected, runner.Issues)

			if tc.fixed != "" {
				t.Logf("Proposed changes: %s", string(runner.Changes()[fileName]))
				helper.AssertChanges(t, map[string]string{fileName: tc.fixed}, runner.Changes())
			} else {
				assert.Empty(t, runner.Changes())
			}
		})
	}
}

func setExpectedRule(expected helper.Issues, rule *MSKTopicConfigRule) {
	for _, exp := range expected {
		exp.Rule = rule
	}
}
