package sharding

type ShardRouter struct {
	ShardCount int // Number of shards
}

func NewShardRouter(shardCount int) *ShardRouter {
	return &ShardRouter{ShardCount: shardCount}
}

func (r *ShardRouter) GetShard(id int) int {
	// Hash the ID and get the shard index
	shardIndex := id % r.ShardCount
	return shardIndex
}
