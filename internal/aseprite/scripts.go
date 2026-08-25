package aseprite

// Lua script bodies. Each runs inside the pcall wrapper in prelude, reads its
// arguments from the table P and returns a table that becomes the tool result.

const luaHealth = `
return {
  version = tostring(app.version),
  apiVersion = app.apiVersion,
  exePath = app.fs.appPath,
}
`

const luaCreateSprite = `
local modes = { rgb = ColorMode.RGB, gray = ColorMode.GRAY, indexed = ColorMode.INDEXED }
local mode = P.colorMode
if mode == nil or mode == "" then mode = "rgb" end
local cm = modes[mode]
if cm == nil then error("colorMode must be rgb, gray or indexed, got: " .. tostring(mode), 0) end
if P.width < 1 or P.height < 1 then error("width and height must be positive", 0) end

local dir = app.fs.filePath(P.path)
if dir ~= "" and not app.fs.isDirectory(dir) then
  error("directory does not exist: " .. dir, 0)
end
if app.fs.isFile(P.path) and not P.overwrite then
  error("file already exists: " .. P.path .. " (pass overwrite=true to replace it)", 0)
end

local sprite = Sprite(P.width, P.height, cm)
sprite:saveAs(P.path)
if not app.fs.isFile(P.path) then error("Aseprite did not write " .. P.path, 0) end

return {
  path = P.path,
  width = sprite.width,
  height = sprite.height,
  colorMode = mode,
}
`

const luaSpriteInfo = `
if not app.fs.isFile(P.path) then error("file not found: " .. P.path, 0) end
local sprite = Sprite{ fromFile = P.path }
if sprite == nil then error("could not open sprite: " .. P.path, 0) end

local modeNames = {
  [ColorMode.RGB] = "rgb",
  [ColorMode.GRAY] = "gray",
  [ColorMode.INDEXED] = "indexed",
  [ColorMode.TILEMAP] = "tilemap",
}

local function describeLayer(layer)
  local t = {
    name = layer.name,
    opacity = layer.opacity,
    visible = layer.isVisible,
    editable = layer.isEditable,
    isGroup = layer.isGroup,
  }
  if layer.isGroup then
    local children = {}
    for i, child in ipairs(layer.layers) do children[i] = describeLayer(child) end
    t.layers = children
    t.layerCount = #children
  else
    t.blendMode = layer.blendMode
    t.celCount = #layer.cels
  end
  return t
end

local layers = {}
for i, layer in ipairs(sprite.layers) do layers[i] = describeLayer(layer) end

local frames = {}
local totalDuration = 0
for i, frame in ipairs(sprite.frames) do
  frames[i] = { number = frame.frameNumber, durationMs = math.floor(frame.duration * 1000 + 0.5) }
  totalDuration = totalDuration + frame.duration
end

local tags = {}
for i, tag in ipairs(sprite.tags) do
  tags[i] = {
    name = tag.name,
    from = tag.fromFrame.frameNumber,
    to = tag.toFrame.frameNumber,
    aniDir = tag.aniDir,
  }
end

-- A 256-entry palette is common and mostly noise, so only a prefix is listed.
local paletteLimit = 32
local palette = sprite.palettes[1]
local shown = math.min(#palette, paletteLimit)
local colors = {}
for i = 0, shown - 1 do
  local c = palette:getColor(i)
  colors[i + 1] = string.format("#%02X%02X%02X%02X", c.red, c.green, c.blue, c.alpha)
end

return {
  path = sprite.filename,
  width = sprite.width,
  height = sprite.height,
  colorMode = modeNames[sprite.colorMode] or tostring(sprite.colorMode),
  frameCount = #sprite.frames,
  layerCount = #sprite.layers,
  tagCount = #sprite.tags,
  totalDurationMs = math.floor(totalDuration * 1000 + 0.5),
  layers = layers,
  frames = frames,
  tags = tags,
  paletteSize = #palette,
  palette = colors,
  paletteTruncated = shown < #palette,
}
`

const luaSaveSpriteAs = `
if not app.fs.isFile(P.source) then error("source not found: " .. P.source, 0) end

local dir = app.fs.filePath(P.destination)
if dir ~= "" and not app.fs.isDirectory(dir) then
  error("directory does not exist: " .. dir, 0)
end

local sprite = Sprite{ fromFile = P.source }
if sprite == nil then error("could not open sprite: " .. P.source, 0) end

-- A single-image format cannot hold an animation, so Aseprite ignores the
-- requested name and writes <title><frame>.<ext> for each frame instead.
-- Those names have to be part of the overwrite check as well.
local base = app.fs.filePathAndTitle(P.destination)
local ext = app.fs.fileExtension(P.destination)
local targets = { P.destination }
if #sprite.frames > 1 then
  for i = 1, #sprite.frames do targets[#targets + 1] = base .. i .. "." .. ext end
end

if not P.overwrite then
  for _, target in ipairs(targets) do
    if app.fs.isFile(target) then
      error("file already exists: " .. target .. " (pass overwrite=true to replace it)", 0)
    end
  end
end

sprite:saveCopyAs(P.destination)

local written = {}
for _, target in ipairs(targets) do
  if app.fs.isFile(target) then written[#written + 1] = target end
end
if #written == 0 then
  error("Aseprite wrote nothing for " .. P.destination .. " (unsupported format?)", 0)
end

return {
  source = P.source,
  destination = P.destination,
  frameCount = #sprite.frames,
  files = written,
  splitIntoSequence = written[1] ~= P.destination,
}
`
