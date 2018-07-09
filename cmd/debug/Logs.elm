module Logs exposing (getLogs)

import Http
import Json.Decode exposing (decodeString, list, string)


getLogs : Http.Request (List String)
getLogs =
    Http.get "http://localhost:4242/logs" (list string)
