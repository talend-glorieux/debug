module Model exposing (..)

import Array exposing (Array)
import Http
import Json.Decode exposing (Decoder, array, string)
import Json.Decode.Pipeline exposing (decode, optional, required)


type alias Model =
    { logs : String
    , services : Array Service
    , selectedService : Int
    }


type alias Service =
    { name : String
    , state : String
    , health : String
    }


getServices : Http.Request (Array Service)
getServices =
    Http.get "http://localhost:4242/services" decodeServices


decodeServices : Decoder (Array Service)
decodeServices =
    array decodeService


decodeService : Decoder Service
decodeService =
    decode Service
        |> required "name" string
        |> required "state" string
        |> optional "health" string ""
