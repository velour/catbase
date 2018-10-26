file = 'example_config.json'
json = require 'cjson'
util = require 'cjson.util'
obj = json.decode(util.file_load(file))
require 'pl.pretty'.dump(obj)