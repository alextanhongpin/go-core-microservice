package singleflight

import redis "github.com/redis/go-redis/v9"

// unlock deletes the key only if the lease id matches.
var unlock = redis.NewScript(`
	-- KEYS[1]: The idempotency key
	-- ARGV[1]: The value value for optimistic locking
	local key = KEYS[1]
	local val = ARGV[1]

	if redis.call('GET', key) == val then
		return redis.call('DEL', key)
	end

	return nil
`)

// replace sets the value to the key only if the existing lease id matches.
var replace = redis.NewScript(`
	-- KEYS[1]: The idempotency key
	-- ARGV[1]: The old value for optimisic locking
	-- ARGV[2]: The new value
	-- ARGV[3]: How long to keep the idempotency key-value pair
	local key = KEYS[1]
	local old = ARGV[1]
	local new = ARGV[2]
	local ttl = ARGV[3]

	if redis.call('GET', key) == old then
		return redis.call('SET', key, new, 'XX', 'PX', ttl) 
	end

	return nil
`)

// extend extends the lock duration only if the lease id matches.
var extend = redis.NewScript(`
	-- KEYS[1]: key
	-- ARGV[1]: value
	-- ARGV[2]: lock duration in milliseconds
	local key = KEYS[1]
	local val = ARGV[1]
	local ttl_ms = tonumber(ARGV[2]) or 60000 -- Default 60s

	if redis.call('GET', key) == val then
		return redis.call('PEXPIRE', key, ttl_ms, 'GT')
	end

	return nil
`)
