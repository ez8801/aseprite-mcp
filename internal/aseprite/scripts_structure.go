package aseprite

const luaAddLayer = `
local sprite = openSprite(P.path)

local layer
if P.group then
  layer = sprite:newGroup()
else
  layer = sprite:newLayer()
end
layer.name = P.name

if P.parent ~= nil and P.parent ~= "" then
  local parent = findLayer(sprite, P.parent)
  if not parent.isGroup then
    error("parent layer '" .. parent.name .. "' is not a group", 0)
  end
  layer.parent = parent
end
if P.opacity ~= nil then layer.opacity = whole(P.opacity) end
if P.visible ~= nil then layer.isVisible = P.visible end

sprite:saveAs(P.path)
return {
  path = P.path,
  name = layer.name,
  isGroup = layer.isGroup,
  parent = P.parent,
  layerCount = #sprite.layers,
}
`

const luaUpdateLayer = `
local sprite = openSprite(P.path)
local layer = findLayer(sprite, P.name)

local changed = {}
if P.newName ~= nil and P.newName ~= "" then
  layer.name = P.newName
  changed[#changed + 1] = "name"
end
if P.opacity ~= nil then
  layer.opacity = whole(P.opacity)
  changed[#changed + 1] = "opacity"
end
if P.visible ~= nil then
  layer.isVisible = P.visible
  changed[#changed + 1] = "visible"
end
if P.blendMode ~= nil and P.blendMode ~= "" then
  if layer.isGroup then error("a group layer has no blend mode", 0) end
  layer.blendMode = toBlendMode(P.blendMode)
  changed[#changed + 1] = "blendMode"
end
if #changed == 0 then error("nothing to update, pass at least one property", 0) end

sprite:saveAs(P.path)
return { path = P.path, name = layer.name, changed = changed }
`

const luaDeleteLayer = `
local sprite = openSprite(P.path)
local layer = findLayer(sprite, P.name)
if #sprite.layers == 1 and layer.parent == sprite then
  error("cannot delete the only top-level layer", 0)
end

local wasGroup = layer.isGroup
sprite:deleteLayer(layer)

sprite:saveAs(P.path)
return { path = P.path, deleted = P.name, wasGroup = wasGroup, layerCount = #sprite.layers }
`

const luaAddFrames = `
local sprite = openSprite(P.path)
local count = whole(P.count or 1)
if count < 1 then error("count must be positive", 0) end

local before = #sprite.frames
local after = P.after
if after ~= nil then after = requireFrame(sprite, after) end

local added = {}
for i = 1, count do
  local frame
  if after == nil then
    -- Appending: newFrame duplicates the last frame, newEmptyFrame does not.
    frame = P.empty and sprite:newEmptyFrame(#sprite.frames + 1) or sprite:newFrame()
  else
    local at = after + i
    frame = P.empty and sprite:newEmptyFrame(at) or sprite:newFrame(at)
  end
  added[#added + 1] = frame.frameNumber
end

sprite:saveAs(P.path)
return {
  path = P.path,
  added = added,
  frameCount = #sprite.frames,
  previousFrameCount = before,
}
`

const luaDeleteFrames = `
local sprite = openSprite(P.path)
if #P.frames == 0 then error("frames must not be empty", 0) end

-- Sort descending so earlier deletions do not shift the remaining numbers.
local wanted = {}
local seen = {}
for _, n in ipairs(P.frames) do
  requireFrame(sprite, n)
  if not seen[n] then
    seen[n] = true
    wanted[#wanted + 1] = n
  end
end
table.sort(wanted, function(a, b) return a > b end)

if #wanted >= #sprite.frames then
  error("cannot delete every frame, a sprite needs at least one", 0)
end
for _, n in ipairs(wanted) do sprite:deleteFrame(n) end

sprite:saveAs(P.path)
return { path = P.path, deleted = wanted, frameCount = #sprite.frames }
`

const luaSetFrameDurations = `
local sprite = openSprite(P.path)
if #P.durations == 0 then error("durations must not be empty", 0) end

local updated = {}
for _, entry in ipairs(P.durations) do
  local n = requireFrame(sprite, entry.frame)
  if entry.durationMs == nil or entry.durationMs < 1 then
    error("frame " .. n .. " needs a positive durationMs", 0)
  end
  local ms = whole(entry.durationMs)
  sprite.frames[n].duration = ms / 1000
  updated[#updated + 1] = { frame = n, durationMs = ms }
end

sprite:saveAs(P.path)
return { path = P.path, updated = updated }
`

const luaSetTag = `
local directions = {
  forward = AniDir.FORWARD,
  reverse = AniDir.REVERSE,
  ping_pong = AniDir.PING_PONG,
  ping_pong_reverse = AniDir.PING_PONG_REVERSE,
}

local sprite = openSprite(P.path)
local from = requireFrame(sprite, P.from)
local to = requireFrame(sprite, P.to)
if to < from then error("to must not be before from", 0) end

-- Aseprite has no way to move a tag's range, so replacing means deleting first.
local existing
for _, tag in ipairs(sprite.tags) do
  if tag.name == P.name then existing = tag end
end
local replaced = existing ~= nil
if replaced then sprite:deleteTag(existing) end

local tag = sprite:newTag(from, to)
tag.name = P.name
if P.aniDir ~= nil and P.aniDir ~= "" then
  local dir = directions[P.aniDir]
  if dir == nil then error("unknown aniDir: " .. P.aniDir, 0) end
  tag.aniDir = dir
end
if P.repeats ~= nil then tag.repeats = whole(P.repeats) end

sprite:saveAs(P.path)
return { path = P.path, name = tag.name, from = from, to = to, replaced = replaced }
`

const luaDeleteTag = `
local sprite = openSprite(P.path)

local existing
for _, tag in ipairs(sprite.tags) do
  if tag.name == P.name then existing = tag end
end
if existing == nil then error("tag not found: " .. P.name, 0) end
sprite:deleteTag(existing)

sprite:saveAs(P.path)
return { path = P.path, deleted = P.name, tagCount = #sprite.tags }
`
