module Main exposing (Model, Msg, init, subscriptions, update, view)

import Html exposing (..)
import Http
import Service exposing (..)
import WebSocket


main : Program Never Model Msg
main =
    Html.program
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }


type alias Model =
    { logs : String
    , services : List Service
    }


type Msg
    = NewServices (Result Http.Error (List Service))
    | Update String


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        NewServices (Ok newServices) ->
            ( { model | services = newServices }, Cmd.none )

        NewServices (Err _) ->
            ( model, Cmd.none )

        Update msg ->
            ( model, Cmd.none )


view : Model -> Html Msg
view model =
    main_ []
        [ nav []
            [ model.services
                |> List.map (\s -> li [] [ text s.name ])
                |> ul []
            ]
        , pre [] [ text model.logs ]
        ]


subscriptions : Model -> Sub Msg
subscriptions model =
    WebSocket.listen "ws://localhost:4242/ws" decodeServices


init : ( Model, Cmd Msg )
init =
    ( Model "" [], Http.send NewServices getServices )
