module Main exposing (init, subscriptions, view)

import Array exposing (..)
import Html exposing (..)
import Html.Attributes exposing (class, id)
import Html.Events exposing (onClick)
import Http
import Json.Decode exposing (..)
import Model exposing (..)
import Update exposing (..)
import WebSocket


main : Program Never Model Msg
main =
    Html.program
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }


view : Model -> Html Msg
view model =
    main_ []
        [ nav [ class "services-list" ] [ viewServices model.services ]
        , pre [ class "logs", id "logs" ] [ text model.logs ]
        ]


viewServices : Array Service -> Html Msg
viewServices services =
    toList services
        |> List.indexedMap (\index service -> li [ class (getServiceClass service.health) ] (viewService index service))
        |> ul []


getServiceClass : String -> String
getServiceClass health =
    if String.isEmpty health then
        "service"
    else
        "service service-" ++ health


viewService : Int -> Service -> List (Html Msg)
viewService index service =
    [ button [ onClick (SelectService index) ] [ h1 [] [ text service.name ] ]
    , h2 [] [ text service.state ]
    , h2 [] [ text service.health ]
    ]


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ WebSocket.listen "ws://localhost:4242/ws" decodeServicesFromString
        , WebSocket.keepAlive "ws://localhost:4242/logs"
        , WebSocket.listen "ws://localhost:4242/logs" LogsStream
        ]


decodeServicesFromString : String -> Msg
decodeServicesFromString msg =
    case decodeString decodeServices msg of
        Err msg ->
            ServicesUpdateError

        Ok msg ->
            ServicesUpdate msg


init : ( Model, Cmd Msg )
init =
    ( Model "" Array.empty -1, Http.send NewServices getServices )
