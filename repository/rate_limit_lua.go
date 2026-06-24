package repository


const tokenBucketLua = `
local capacity = tonumber(ARGV[1])
local period_sec = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local refill_per_ms = capacity / (period_sec * 1000)

local data = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])

if tokens == nil then
  tokens = capacity
  ts = now_ms
end

local elapsed = now_ms - ts
if elapsed > 0 then
  tokens = math.min(capacity, tokens + elapsed * refill_per_ms)
  ts = now_ms
end

if tokens < cost then
  local deficit = cost - tokens
  local wait_ms = deficit / refill_per_ms
  local retry_sec = math.ceil(wait_ms / 1000)
  if retry_sec < 1 then
    retry_sec = 1
  end
  return {0, retry_sec}
end

tokens = tokens - cost
redis.call('HMSET', KEYS[1], 'tokens', tokens, 'ts', ts)
redis.call('EXPIRE', KEYS[1], period_sec * 2)
return {1, 0}
`
