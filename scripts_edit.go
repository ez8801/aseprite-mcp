package main

// luaHelpers is spliced into every generated script by prelude. It must not
// contain percent verbs, since prelude is a format string.
const luaHelpers = `
-- Numbers decoded from JSON arrive as floats, so they are rounded before being
-- used as counts, indexes or coordinates.
local function whole(v)
  return math.floor(v + 0.5)
end

local function openSprite(path)
  if not app.fs.isFile(path) then error("file not found: " .. path, 0) end
  local sprite = Sprite{ fromFile = path }
  if sprite == nil then error("could not open sprite: " .. path, 0) end
  return sprite
end

-- findLayer walks groups too. Aseprite allows duplicate names; the first match
-- in stacking order wins.
local function findLayer(sprite, name)
  local function search(layers)
    for _, layer in ipairs(layers) do
      if layer.name == name then return layer end
      if layer.isGroup then
        local found = search(layer.layers)
        if found ~= nil then return found end
      end
    end
    return nil
  end
  local found = search(sprite.layers)
  if found == nil then error("layer not found: " .. name, 0) end
  return found
end

-- firstImageLayer is the fallback target when no layer name is given.
local function firstImageLayer(sprite)
  local function search(layers)
    for _, layer in ipairs(layers) do
      if layer.isImage then return layer end
      if layer.isGroup then
        local found = search(layer.layers)
        if found ~= nil then return found end
      end
    end
    return nil
  end
  local found = search(sprite.layers)
  if found == nil then error("sprite has no image layer to draw on", 0) end
  return found
end

-- drawTarget resolves the layer to paint on. Drawing on a group silently does
-- nothing in Aseprite, so groups are rejected up front.
local function drawTarget(sprite, name)
  local layer
  if name == nil or name == "" then
    layer = firstImageLayer(sprite)
  else
    layer = findLayer(sprite, name)
  end
  if not layer.isImage then
    error("layer '" .. layer.name .. "' is a group and cannot hold pixels", 0)
  end
  return layer
end

local function requireFrame(sprite, number)
  local n = whole(number or 1)
  if n < 1 or n > #sprite.frames then
    error("frame " .. n .. " is out of range, sprite has " .. #sprite.frames .. " frame(s)", 0)
  end
  return n
end

-- parseColor accepts "#RRGGBB" or "#RRGGBBAA" for rgb and gray sprites, and a
-- bare number for indexed sprites, where it is used as a palette index.
local function parseColor(spec)
  if type(spec) == "number" then return math.floor(spec) end
  if type(spec) ~= "string" then
    error("color must be a hex string like #RRGGBB or a palette index", 0)
  end
  if spec:sub(1, 1) ~= "#" then
    local index = tonumber(spec)
    if index == nil then
      error("color must start with # or be a palette index, got: " .. spec, 0)
    end
    return math.floor(index)
  end
  local hex = spec:sub(2)
  if #hex == 6 then hex = hex .. "FF" end
  if #hex ~= 8 then
    error("color must have 6 or 8 hex digits, got: " .. spec, 0)
  end
  local r = tonumber(hex:sub(1, 2), 16)
  local g = tonumber(hex:sub(3, 4), 16)
  local b = tonumber(hex:sub(5, 6), 16)
  local a = tonumber(hex:sub(7, 8), 16)
  if r == nil or g == nil or b == nil or a == nil then
    error("color is not valid hex: " .. spec, 0)
  end
  return Color{ r = r, g = g, b = b, a = a }
end

local blendModes = {
  normal = BlendMode.NORMAL,
  multiply = BlendMode.MULTIPLY,
  screen = BlendMode.SCREEN,
  overlay = BlendMode.OVERLAY,
  darken = BlendMode.DARKEN,
  lighten = BlendMode.LIGHTEN,
  color_dodge = BlendMode.COLOR_DODGE,
  color_burn = BlendMode.COLOR_BURN,
  hard_light = BlendMode.HARD_LIGHT,
  soft_light = BlendMode.SOFT_LIGHT,
  difference = BlendMode.DIFFERENCE,
  exclusion = BlendMode.EXCLUSION,
  hue = BlendMode.HUE,
  saturation = BlendMode.SATURATION,
  color = BlendMode.COLOR,
  luminosity = BlendMode.LUMINOSITY,
  addition = BlendMode.ADDITION,
  subtract = BlendMode.SUBTRACT,
  divide = BlendMode.DIVIDE,
}

local function toBlendMode(name)
  if name == nil or name == "" then return BlendMode.NORMAL end
  local mode = blendModes[name]
  if mode == nil then error("unknown blendMode: " .. name, 0) end
  return mode
end

-- json.decode returns userdata proxies rather than Lua tables, so a nested
-- value is probed by field instead of by type.
local function toPoint(p, label)
  if p == nil or type(p.x) ~= "number" or type(p.y) ~= "number" then
    error(label .. " must be an object with numeric x and y", 0)
  end
  return Point(whole(p.x), whole(p.y))
end
`

const luaDrawPixels = `
local sprite = openSprite(P.path)
local layer = drawTarget(sprite, P.layer)
local frame = requireFrame(sprite, P.frame)
if #P.pixels == 0 then error("pixels must not be empty", 0) end

-- Passing several points to one useTool call draws a stroke that connects
-- them, so each pixel needs a call of its own.
for i = 1, #P.pixels do
  local spec = P.pixels[i]
  app.useTool{
    tool = "pencil",
    color = parseColor(spec.color),
    points = { toPoint(spec, "pixel") },
    sprite = sprite,
    layer = layer,
    frame = frame,
  }
end

sprite:saveAs(P.path)
return { path = P.path, layer = layer.name, frame = frame, pixels = #P.pixels }
`

const luaDrawShapes = `
local tools = {
  line = "line",
  rectangle = "rectangle",
  filled_rectangle = "filled_rectangle",
  ellipse = "ellipse",
  filled_ellipse = "filled_ellipse",
  contour = "contour",
}

local sprite = openSprite(P.path)
local layer = drawTarget(sprite, P.layer)
local frame = requireFrame(sprite, P.frame)
if #P.shapes == 0 then error("shapes must not be empty", 0) end

for i, shape in ipairs(P.shapes) do
  local tool = tools[shape.kind]
  if tool == nil then error("shape " .. i .. " has unknown kind: " .. tostring(shape.kind), 0) end
  local args = {
    tool = tool,
    color = parseColor(shape.color),
    points = { toPoint(shape.from, "shape.from"), toPoint(shape.to, "shape.to") },
    sprite = sprite,
    layer = layer,
    frame = frame,
  }
  if shape.brushSize ~= nil and shape.brushSize > 1 then
    args.brush = Brush(whole(shape.brushSize))
  end
  app.useTool(args)
end

sprite:saveAs(P.path)
return { path = P.path, layer = layer.name, frame = frame, shapes = #P.shapes }
`

const luaFillArea = `
local sprite = openSprite(P.path)
local layer = drawTarget(sprite, P.layer)
local frame = requireFrame(sprite, P.frame)

local x, y = whole(P.x), whole(P.y)
app.useTool{
  tool = "paint_bucket",
  color = parseColor(P.color),
  points = { Point(x, y) },
  sprite = sprite,
  layer = layer,
  frame = frame,
  tolerance = whole(P.tolerance or 0),
  contiguous = P.contiguous ~= false,
}

sprite:saveAs(P.path)
return { path = P.path, layer = layer.name, frame = frame, x = x, y = y }
`

// Aseprite quantizes an rgb image against a default palette rather than the
// destination's, so anything but an rgb destination is converted by hand.
const luaConverters = `
local function converters(sprite)
  if sprite.colorMode == ColorMode.GRAY then
    local function toRGB(v)
      local level = app.pixelColor.grayaV(v)
      return app.pixelColor.rgba(level, level, level, app.pixelColor.grayaA(v))
    end
    local function fromRGB(v)
      local alpha = app.pixelColor.rgbaA(v)
      if alpha == 0 then return 0 end
      local level = (app.pixelColor.rgbaR(v) * 30 + app.pixelColor.rgbaG(v) * 59
                     + app.pixelColor.rgbaB(v) * 11) // 100
      return app.pixelColor.graya(level, alpha)
    end
    return toRGB, fromRGB
  end

  local palette = sprite.palettes[1]
  local size = #palette
  local transparent = sprite.transparentColor
  local entries = {}
  for i = 0, size - 1 do
    local c = palette:getColor(i)
    entries[i] = { r = c.red, g = c.green, b = c.blue, a = c.alpha }
  end

  local function toRGB(v)
    local entry = entries[v]
    if entry == nil or v == transparent then return 0 end
    return app.pixelColor.rgba(entry.r, entry.g, entry.b, entry.a)
  end

  -- Pixel art repeats few colors, so caching the search keeps this linear.
  local cache = {}
  local function fromRGB(v)
    local hit = cache[v]
    if hit ~= nil then return hit end
    local best = transparent
    if app.pixelColor.rgbaA(v) > 0 then
      local r = app.pixelColor.rgbaR(v)
      local g = app.pixelColor.rgbaG(v)
      local b = app.pixelColor.rgbaB(v)
      local bestDistance
      for i = 0, size - 1 do
        local entry = entries[i]
        if entry.a > 0 then
          local dr, dg, db = r - entry.r, g - entry.g, b - entry.b
          local distance = dr * dr + dg * dg + db * db
          if bestDistance == nil or distance < bestDistance then
            best, bestDistance = i, distance
          end
        end
      end
    end
    cache[v] = best
    return best
  end
  return toRGB, fromRGB
end
`

const luaStampSprites = luaConverters + `
local dest = openSprite(P.destination)
local layer = drawTarget(dest, P.layer)
local frame = requireFrame(dest, P.frame)
if #P.stamps == 0 then error("stamps must not be empty", 0) end

local destRect = Rectangle(0, 0, dest.width, dest.height)

-- An rgb destination can be composited with Aseprite's own blending. Any other
-- color mode goes through rgb and is converted back a pixel at a time.
local directRGB = dest.colorMode == ColorMode.RGB
local toRGB, fromRGB
if not directRGB then toRGB, fromRGB = converters(dest) end

-- Reopening the same file for every stamp would be wasteful when one source is
-- placed several times.
local opened = {}
local function sourceSprite(path)
  if opened[path] == nil then
    if not app.fs.isFile(path) then error("source not found: " .. path, 0) end
    local sprite = Sprite{ fromFile = path }
    if sprite == nil then error("could not open sprite: " .. path, 0) end
    opened[path] = sprite
  end
  return opened[path]
end

local placed = {}
for i, stamp in ipairs(P.stamps) do
  local source = sourceSprite(stamp.source)
  local sourceFrame = whole(stamp.sourceFrame or 1)
  if sourceFrame < 1 or sourceFrame > #source.frames then
    error("stamp " .. i .. ": sourceFrame " .. sourceFrame .. " is out of range, the source has "
          .. #source.frames .. " frame(s)", 0)
  end

  local x, y = whole(stamp.x), whole(stamp.y)
  local visible = Rectangle(x, y, source.width, source.height):intersect(destRect)
  if visible.width < 1 or visible.height < 1 then
    error("stamp " .. i .. ": lands entirely outside the destination sprite", 0)
  end

  -- Flattening to rgb first is what keeps an indexed destination mapping
  -- through its own palette; drawing a sprite straight into an indexed image
  -- picks the wrong indexes.
  local flat = Image(source.width, source.height, ColorMode.RGB)
  flat:drawSprite(source, sourceFrame)

  -- Opening the source made it the active sprite, and an indexed destination
  -- is quantized against whichever sprite is active, so point it back.
  app.sprite = dest

  local cel = layer:cel(frame)
  local bounds = visible
  if cel ~= nil then bounds = bounds:union(cel.bounds) end

  local function newImage(width, height)
    return Image(ImageSpec{
      width = width,
      height = height,
      colorMode = dest.colorMode,
      transparentColor = dest.transparentColor,
    })
  end

  local opacity = whole(stamp.opacity or 255)
  local blend = toBlendMode(stamp.blendMode)

  local canvas
  if directRGB then
    canvas = newImage(bounds.width, bounds.height)
    if cel ~= nil then
      canvas:drawImage(cel.image, Point(cel.bounds.x - bounds.x, cel.bounds.y - bounds.y))
    end
    canvas:drawImage(flat, Point(x - bounds.x, y - bounds.y), opacity, blend)
  else
    local staging = Image(bounds.width, bounds.height, ColorMode.RGB)
    if cel ~= nil then
      local dx, dy = cel.bounds.x - bounds.x, cel.bounds.y - bounds.y
      for py = 0, cel.bounds.height - 1 do
        for px = 0, cel.bounds.width - 1 do
          staging:drawPixel(px + dx, py + dy, toRGB(cel.image:getPixel(px, py)))
        end
      end
    end
    staging:drawImage(flat, Point(x - bounds.x, y - bounds.y), opacity, blend)

    canvas = newImage(bounds.width, bounds.height)
    for py = 0, bounds.height - 1 do
      for px = 0, bounds.width - 1 do
        canvas:drawPixel(px, py, fromRGB(staging:getPixel(px, py)))
      end
    end
  end

  -- A new sprite already carries a cel the size of the whole canvas, so the
  -- union above would keep every cel full size. Trim back to real content.
  local tight = canvas:shrinkBounds()
  if tight.width > 0 and tight.height > 0
     and (tight.width < bounds.width or tight.height < bounds.height) then
    local cropped = newImage(tight.width, tight.height)
    cropped:drawImage(canvas, Point(-tight.x, -tight.y))
    canvas = cropped
    bounds = Rectangle(bounds.x + tight.x, bounds.y + tight.y, tight.width, tight.height)
  end

  if cel ~= nil then
    cel.image = canvas
    cel.position = Point(bounds.x, bounds.y)
  else
    dest:newCel(layer, frame, canvas, Point(bounds.x, bounds.y))
  end

  placed[#placed + 1] = {
    source = stamp.source,
    x = x,
    y = y,
    width = source.width,
    height = source.height,
    clipped = visible.width < source.width or visible.height < source.height,
  }
end

dest:saveAs(P.destination)
return { destination = P.destination, layer = layer.name, frame = frame, stamps = placed }
`

// app.command.Clear crashes Aseprite in batch mode whenever the active cel is
// missing, so clearing goes through the cel image instead.
const luaClearArea = `
local sprite = openSprite(P.path)
local layer = drawTarget(sprite, P.layer)
local frame = requireFrame(sprite, P.frame)

local cel = layer:cel(frame)
local cleared = "nothing"
if cel == nil then
  cleared = "empty"
elseif P.rect == nil then
  sprite:deleteCel(layer, frame)
  cleared = "cel"
else
  local r = P.rect
  if whole(r.width) < 1 or whole(r.height) < 1 then error("rect width and height must be positive", 0) end
  -- Image coordinates are relative to the cel, not to the sprite.
  cel.image:clear(Rectangle(whole(r.x) - cel.bounds.x, whole(r.y) - cel.bounds.y, whole(r.width), whole(r.height)))
  cleared = "rect"
end

sprite:saveAs(P.path)
return { path = P.path, layer = layer.name, frame = frame, cleared = cleared }
`
