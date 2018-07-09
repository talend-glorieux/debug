module Service exposing (Service, getServices)

import Http
import Json.Decode exposing (Decoder, list, string)
import Json.Decode.Pipeline exposing (decode, optional, required)


type alias Service =
    { name : String
    , state : String
    , health : String
    }


getServices : Http.Request (List Service)
getServices =
    Http.get "http://localhost:4242/services" (list decodeService)


decodeService : Decoder Service
decodeService =
    decode Service
        |> required "name" string
        |> required "state" string
        |> optional "health" string ""
