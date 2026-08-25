package main

const luaSetPalette = `
local sprite = openSprite(P.path)

local palette
local source
if P.fromFile ~= nil and P.fromFile ~= "" then
  if not app.fs.isFile(P.fromFile) then error("palette file not found: " .. P.fromFile, 0) end
  palette = Palette{ fromFile = P.fromFile }
  if palette == nil then error("could not read palette: " .. P.fromFile, 0) end
  source = P.fromFile
else
  if P.colors == nil or #P.colors == 0 then
    error("pass either colors or fromFile", 0)
  end
  palette = Palette(#P.colors)
  for i, spec in ipairs(P.colors) do
    local color = parseColor(spec)
    if type(color) == "number" then
      error("palette colors must be hex strings, not indices: " .. tostring(spec), 0)
    end
    palette:setColor(i - 1, color)
  end
  source = "colors"
end

sprite:setPalette(palette)
sprite:saveAs(P.path)
return { path = P.path, source = source, paletteSize = #sprite.palettes[1] }
`

const luaSavePalette = `
local sprite = openSprite(P.path)

local dir = app.fs.filePath(P.destination)
if dir ~= "" and not app.fs.isDirectory(dir) then
  error("directory does not exist: " .. dir, 0)
end
if app.fs.isFile(P.destination) and not P.overwrite then
  error("file already exists: " .. P.destination .. " (pass overwrite=true to replace it)", 0)
end

sprite.palettes[1]:saveAs(P.destination)
if not app.fs.isFile(P.destination) then
  error("Aseprite did not write " .. P.destination .. " (unsupported format?)", 0)
end

return { path = P.path, destination = P.destination, paletteSize = #sprite.palettes[1] }
`

const luaGetPalette = `
local sprite = openSprite(P.path)
local palette = sprite.palettes[1]

local colors = {}
for i = 0, #palette - 1 do
  local c = palette:getColor(i)
  colors[i + 1] = string.format("#%02X%02X%02X%02X", c.red, c.green, c.blue, c.alpha)
end

return {
  path = P.path,
  paletteSize = #palette,
  colors = colors,
  transparentIndex = sprite.transparentColor,
}
`

const luaResizeSprite = `
local sprite = openSprite(P.path)

local width, height
if P.scale ~= nil then
  if P.scale <= 0 then error("scale must be positive", 0) end
  width = math.floor(sprite.width * P.scale + 0.5)
  height = math.floor(sprite.height * P.scale + 0.5)
else
  width = whole(P.width or sprite.width)
  height = whole(P.height or sprite.height)
end
if width < 1 or height < 1 then error("resulting size must be at least 1x1", 0) end

local before = { width = sprite.width, height = sprite.height }
sprite:resize(width, height)

sprite:saveAs(P.path)
return { path = P.path, from = before, width = sprite.width, height = sprite.height }
`

// ExportSpriteSheet writes its data JSON to stdout when no dataFilename is
// given, which the sentinel in prelude filters out of the result.
const luaExportSpritesheet = `
local sheetTypes = {
  horizontal = SpriteSheetType.HORIZONTAL,
  vertical = SpriteSheetType.VERTICAL,
  rows = SpriteSheetType.ROWS,
  columns = SpriteSheetType.COLUMNS,
  packed = SpriteSheetType.PACKED,
}

local sprite = openSprite(P.path)

local dir = app.fs.filePath(P.destination)
if dir ~= "" and not app.fs.isDirectory(dir) then
  error("directory does not exist: " .. dir, 0)
end

local sheetType = sheetTypes[P.sheetType or "packed"]
if sheetType == nil then error("unknown sheetType: " .. tostring(P.sheetType), 0) end

local targets = { P.destination }
if P.dataFile ~= nil and P.dataFile ~= "" then targets[#targets + 1] = P.dataFile end
if not P.overwrite then
  for _, target in ipairs(targets) do
    if app.fs.isFile(target) then
      error("file already exists: " .. target .. " (pass overwrite=true to replace it)", 0)
    end
  end
end

local args = {
  ui = false,
  type = sheetType,
  textureFilename = P.destination,
  splitLayers = P.splitLayers == true,
  splitTags = P.splitTags == true,
  shapePadding = whole(P.padding or 0),
  trim = P.trim == true,
}
if P.dataFile ~= nil and P.dataFile ~= "" then
  args.dataFilename = P.dataFile
  args.dataFormat = SpriteSheetDataFormat.JSON_HASH
  if P.dataFormat == "array" then args.dataFormat = SpriteSheetDataFormat.JSON_ARRAY end
end
app.command.ExportSpriteSheet(args)

if not app.fs.isFile(P.destination) then
  error("Aseprite did not write " .. P.destination, 0)
end
if args.dataFilename ~= nil and not app.fs.isFile(args.dataFilename) then
  error("Aseprite did not write " .. args.dataFilename, 0)
end

return {
  path = P.path,
  destination = P.destination,
  dataFile = P.dataFile,
  sheetType = P.sheetType or "packed",
  frameCount = #sprite.frames,
}
`
