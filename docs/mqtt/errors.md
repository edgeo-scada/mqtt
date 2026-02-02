# Gestion des erreurs

Documentation des erreurs et codes de raison MQTT 5.0.

## Erreurs standard

La bibliothèque définit des erreurs standard pour les situations courantes:

```go
var (
    ErrNotConnected        = errors.New("mqtt: not connected")
    ErrAlreadyConnected    = errors.New("mqtt: already connected")
    ErrConnectionLost      = errors.New("mqtt: connection lost")
    ErrTimeout             = errors.New("mqtt: operation timed out")
    ErrInvalidQoS          = errors.New("mqtt: invalid QoS level")
    ErrInvalidTopic        = errors.New("mqtt: invalid topic")
    ErrMalformedPacket     = errors.New("mqtt: malformed packet")
    ErrProtocolError       = errors.New("mqtt: protocol error")
    ErrClientClosed        = errors.New("mqtt: client closed")
)
```

### Vérification des erreurs

```go
token := client.Publish(ctx, topic, payload, mqtt.QoS1, false)
if err := token.Wait(); err != nil {
    switch {
    case errors.Is(err, mqtt.ErrNotConnected):
        log.Println("Client non connecté, tentative de reconnexion...")
    case errors.Is(err, mqtt.ErrTimeout):
        log.Println("Timeout de publication")
    default:
        log.Printf("Erreur: %v", err)
    }
}
```

## Codes de raison MQTT 5.0

MQTT 5.0 utilise des codes de raison détaillés pour chaque opération.

### ReasonCode

```go
type ReasonCode byte
```

### Codes de succès

| Code | Constante | Description |
|------|-----------|-------------|
| 0x00 | `ReasonSuccess` | Succès / QoS 0 accordé |
| 0x01 | `ReasonGrantedQoS1` | QoS 1 accordé |
| 0x02 | `ReasonGrantedQoS2` | QoS 2 accordé |
| 0x04 | `ReasonDisconnectWithWill` | Déconnexion avec Will |
| 0x10 | `ReasonNoMatchingSubscribers` | Pas d'abonnés correspondants |
| 0x11 | `ReasonNoSubscriptionExisted` | Aucun abonnement existant |
| 0x18 | `ReasonContinueAuthentication` | Continuer l'authentification |
| 0x19 | `ReasonReAuthenticate` | Ré-authentification |

### Codes d'erreur

| Code | Constante | Description |
|------|-----------|-------------|
| 0x80 | `ReasonUnspecifiedError` | Erreur non spécifiée |
| 0x81 | `ReasonMalformedPacket` | Paquet malformé |
| 0x82 | `ReasonProtocolError` | Erreur de protocole |
| 0x83 | `ReasonImplementationError` | Erreur d'implémentation |
| 0x84 | `ReasonUnsupportedProtocolVersion` | Version protocole non supportée |
| 0x85 | `ReasonClientIDNotValid` | Client ID invalide |
| 0x86 | `ReasonBadUsernameOrPassword` | Mauvais username/password |
| 0x87 | `ReasonNotAuthorized` | Non autorisé |
| 0x88 | `ReasonServerUnavailable` | Serveur indisponible |
| 0x89 | `ReasonServerBusy` | Serveur occupé |
| 0x8A | `ReasonBanned` | Client banni |
| 0x8B | `ReasonServerShuttingDown` | Serveur en arrêt |
| 0x8C | `ReasonBadAuthenticationMethod` | Mauvaise méthode d'authentification |
| 0x8D | `ReasonKeepAliveTimeout` | Timeout keep-alive |
| 0x8E | `ReasonSessionTakenOver` | Session reprise par un autre client |
| 0x8F | `ReasonTopicFilterInvalid` | Filtre de topic invalide |
| 0x90 | `ReasonTopicNameInvalid` | Nom de topic invalide |
| 0x91 | `ReasonPacketIDInUse` | Packet ID déjà utilisé |
| 0x92 | `ReasonPacketIDNotFound` | Packet ID non trouvé |
| 0x93 | `ReasonReceiveMaximumExceeded` | Receive maximum dépassé |
| 0x94 | `ReasonTopicAliasInvalid` | Topic alias invalide |
| 0x95 | `ReasonPacketTooLarge` | Paquet trop grand |
| 0x96 | `ReasonMessageRateTooHigh` | Débit de messages trop élevé |
| 0x97 | `ReasonQuotaExceeded` | Quota dépassé |
| 0x98 | `ReasonAdministrativeAction` | Action administrative |
| 0x99 | `ReasonPayloadFormatInvalid` | Format de payload invalide |
| 0x9A | `ReasonRetainNotSupported` | Retain non supporté |
| 0x9B | `ReasonQoSNotSupported` | QoS non supporté |
| 0x9C | `ReasonUseAnotherServer` | Utiliser un autre serveur |
| 0x9D | `ReasonServerMoved` | Serveur déplacé |
| 0x9E | `ReasonSharedSubsNotSupported` | Shared subscriptions non supportées |
| 0x9F | `ReasonConnectionRateExceeded` | Taux de connexion dépassé |
| 0xA0 | `ReasonMaxConnectTime` | Temps de connexion maximum |
| 0xA1 | `ReasonSubIDNotSupported` | Subscription ID non supporté |
| 0xA2 | `ReasonWildcardSubsNotSupported` | Wildcards non supportés |

### Méthodes ReasonCode

```go
// Conversion en string
code.String() string

// Vérifie si c'est une erreur
code.IsError() bool

// Convertit en error Go
code.ToError() error
```

## Types d'erreurs

### MQTTError

Erreur MQTT générique avec code de raison.

```go
type MQTTError struct {
    Code       ReasonCode
    Message    string
    Properties *Properties
}

func (e *MQTTError) Error() string
```

**Exemple:**

```go
if err := token.Wait(); err != nil {
    var mqttErr *mqtt.MQTTError
    if errors.As(err, &mqttErr) {
        log.Printf("Erreur MQTT: code=%s, message=%s",
            mqttErr.Code.String(), mqttErr.Message)

        if mqttErr.Properties != nil && mqttErr.Properties.ReasonString != "" {
            log.Printf("Raison: %s", mqttErr.Properties.ReasonString)
        }
    }
}
```

### ConnectError

Erreur de connexion avec détails.

```go
type ConnectError struct {
    Code       ReasonCode
    Properties *Properties
}

func (e *ConnectError) Error() string
```

**Exemple:**

```go
if err := client.Connect(ctx); err != nil {
    var connErr *mqtt.ConnectError
    if errors.As(err, &connErr) {
        switch connErr.Code {
        case mqtt.ReasonBadUsernameOrPassword:
            log.Println("Identifiants incorrects")
        case mqtt.ReasonNotAuthorized:
            log.Println("Accès non autorisé")
        case mqtt.ReasonServerUnavailable:
            log.Println("Serveur indisponible")
        case mqtt.ReasonBanned:
            log.Println("Client banni")
        default:
            log.Printf("Connexion refusée: %s", connErr.Code.String())
        }

        // Vérifier si le serveur suggère une redirection
        if connErr.Properties != nil && connErr.Properties.ServerReference != "" {
            log.Printf("Redirection suggérée: %s", connErr.Properties.ServerReference)
        }
    }
}
```

### DisconnectError

Erreur de déconnexion initiée par le serveur.

```go
type DisconnectError struct {
    Code       ReasonCode
    Properties *Properties
}

func (e *DisconnectError) Error() string
```

**Exemple:**

```go
client := mqtt.NewClient(
    mqtt.WithOnConnectionLost(func(c *mqtt.Client, err error) {
        var discErr *mqtt.DisconnectError
        if errors.As(err, &discErr) {
            switch discErr.Code {
            case mqtt.ReasonSessionTakenOver:
                log.Println("Session reprise par un autre client")
            case mqtt.ReasonKeepAliveTimeout:
                log.Println("Timeout keep-alive")
            case mqtt.ReasonAdministrativeAction:
                log.Println("Déconnecté par l'administrateur")
            case mqtt.ReasonServerShuttingDown:
                log.Println("Serveur en arrêt")
            default:
                log.Printf("Déconnecté: %s", discErr.Code.String())
            }
        }
    }),
)
```

## Gestion des erreurs par opération

### Connexion

```go
func connectWithRetry(client *mqtt.Client) error {
    backoff := time.Second

    for i := 0; i < 5; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        err := client.Connect(ctx)
        cancel()

        if err == nil {
            return nil
        }

        var connErr *mqtt.ConnectError
        if errors.As(err, &connErr) {
            // Erreurs non récupérables
            switch connErr.Code {
            case mqtt.ReasonBanned,
                 mqtt.ReasonBadUsernameOrPassword,
                 mqtt.ReasonNotAuthorized:
                return err // Ne pas réessayer
            }
        }

        log.Printf("Tentative %d échouée: %v", i+1, err)
        time.Sleep(backoff)
        backoff *= 2
    }

    return errors.New("connexion impossible après 5 tentatives")
}
```

### Publication

```go
func publish(client *mqtt.Client, topic string, payload []byte) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    token := client.Publish(ctx, topic, payload, mqtt.QoS1, false)

    select {
    case <-token.Done():
        if err := token.Error(); err != nil {
            var mqttErr *mqtt.MQTTError
            if errors.As(err, &mqttErr) {
                switch mqttErr.Code {
                case mqtt.ReasonPacketTooLarge:
                    return fmt.Errorf("message trop grand pour le broker")
                case mqtt.ReasonQuotaExceeded:
                    return fmt.Errorf("quota dépassé, réessayer plus tard")
                case mqtt.ReasonTopicNameInvalid:
                    return fmt.Errorf("topic invalide: %s", topic)
                }
            }
            return err
        }
        return nil
    case <-ctx.Done():
        return mqtt.ErrTimeout
    }
}
```

### Souscription

```go
func subscribe(client *mqtt.Client, topic string, handler mqtt.MessageHandler) error {
    token := client.Subscribe(context.Background(), topic, mqtt.QoS1, handler)
    if err := token.Wait(); err != nil {
        var mqttErr *mqtt.MQTTError
        if errors.As(err, &mqttErr) {
            switch mqttErr.Code {
            case mqtt.ReasonTopicFilterInvalid:
                return fmt.Errorf("filtre de topic invalide: %s", topic)
            case mqtt.ReasonSharedSubsNotSupported:
                return fmt.Errorf("shared subscriptions non supportées")
            case mqtt.ReasonWildcardSubsNotSupported:
                return fmt.Errorf("wildcards non supportés")
            case mqtt.ReasonSubIDNotSupported:
                return fmt.Errorf("subscription IDs non supportés")
            }
        }
        return err
    }

    // Vérifier le QoS accordé
    if len(token.GrantedQoS) > 0 && token.GrantedQoS[0] < mqtt.QoS1 {
        log.Printf("Attention: QoS demandé non accordé, reçu QoS %d", token.GrantedQoS[0])
    }

    return nil
}
```

## Logging des erreurs

```go
import "log/slog"

func logError(logger *slog.Logger, operation string, err error) {
    var mqttErr *mqtt.MQTTError
    var connErr *mqtt.ConnectError
    var discErr *mqtt.DisconnectError

    switch {
    case errors.As(err, &mqttErr):
        logger.Error("MQTT error",
            "operation", operation,
            "code", mqttErr.Code.String(),
            "code_hex", fmt.Sprintf("0x%02X", byte(mqttErr.Code)),
            "message", mqttErr.Message,
        )
    case errors.As(err, &connErr):
        logger.Error("Connection error",
            "operation", operation,
            "code", connErr.Code.String(),
        )
    case errors.As(err, &discErr):
        logger.Error("Disconnect error",
            "operation", operation,
            "code", discErr.Code.String(),
        )
    default:
        logger.Error("Error",
            "operation", operation,
            "error", err.Error(),
        )
    }
}
```
