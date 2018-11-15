module Update exposing (..)

import Array exposing (Array, get)
import Dom.Scroll
import Http
import Model exposing (Model, Service)
import Task
import WebSocket


type Msg
    = NewServices (Result Http.Error (Array Service))
    | ServicesUpdate (Array Service)
    | ServicesUpdateError
    | LogsStream String
    | SelectService Int
    | NoOp


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        NoOp ->
            ( model, Cmd.none )

        NewServices (Ok newServices) ->
            ( { model | services = newServices }, getLogs (get 0 newServices) )

        NewServices (Err _) ->
            ( model, Cmd.none )

        ServicesUpdate msg ->
            ( { model | services = msg }, Cmd.none )

        ServicesUpdateError ->
            ( model, Cmd.none )

        LogsStream l ->
            ( { model | logs = model.logs ++ l ++ "\n" }, Task.attempt (always NoOp) (Dom.Scroll.toBottom "logs") )

        SelectService index ->
            ( { model | selectedService = index, logs = "" }, getLogs (get index model.services) )


getLogs : Maybe Service -> Cmd msg
getLogs service =
    case service of
        Just s ->
            WebSocket.send "ws://localhost:4242/logs" s.name

        Nothing ->
            Cmd.none
