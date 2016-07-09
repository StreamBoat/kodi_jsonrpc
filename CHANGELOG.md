
<a name="v2.0.6"></a>
# v2.0.6 (2016-07-09)

## :house: Housekeeping

- **core**:
  - Add deprecation notices to this package. ([4d22ed3b](https://github.com/StreamBoat/kodi_jsonrpc/commit/4d22ed3be682cd4de61e928dccbee81b42383028))  
    <br>New package is available at https://github.com/pdf/kodirpc


<a name="v2.0.4"></a>
# v2.0.4 (2015-07-05)

## :bug: Bug Fixes

- **core**:
  - Drop fewer notifications ([22b24707](https://github.com/StreamBoat/kodi_jsonrpc/commit/22b2470773405242524e4a8b93ccc6c9216170dd))  <br>After converting notifications to a ring buffer, we were dropping them
    on notification floods, even for clients that were consuming
    notifications.  The buffer now drops old notifications after a 200ms
    delay to give clients some time to catch up, and notifications are
    processed in their own routine so that this delay doesn't block normal
    responses.  The trade-off is that we lose notification ordering
    guarantees, but I think this is probably acceptable.


<a name="v2.0.3"></a>
# v2.0.3 (2015-07-05)

## :house: Housekeeping

- **logs**:
  - Log notifications by field ([7c613ef3](https://github.com/StreamBoat/kodi_jsonrpc/commit/7c613ef3a0edbe13f33b9060e7cb55be26244588))


<a name="v2.0.2"></a>
# v2.0.2 (2015-07-05)

## :house: Housekeeping

- **logs**:
  - Include full notification content in debug output ([942adb8d](https://github.com/StreamBoat/kodi_jsonrpc/commit/942adb8d38828599524bedf359f18ac17cf20f90))


<a name="v2.0.1"></a>
# v2.0.1 (2015-07-05)

## :house: Housekeeping

- **style**:
  - Fix lint that won't break existing APIs ([54af862a](https://github.com/StreamBoat/kodi_jsonrpc/commit/54af862af1928000f87bcbb3092cf1ac59746960))


<a name="v2.0.0"></a>
# v2.0.0 (2015-03-17)

## :bug: Bug Fixes

- **core**:
  - Implement ring buffer for notification writes ([a9e60882](https://github.com/StreamBoat/kodi_jsonrpc/commit/a9e60882ddab062ca7fced3cb56d5586b9e1ad1f))  <br>Clients that don't process notifications could result in blocked reader
    goroutine

## :house: Housekeeping

- **core**:
  - Return error trying to send on closed connection ([2ad70a41](https://github.com/StreamBoat/kodi_jsonrpc/commit/2ad70a415c661980a3001245d3757ee483461c09))

## Breaking Changes

- due to [2ad70a41](https://github.com/StreamBoat/kodi_jsonrpc/commit/2ad70a415c661980a3001245d3757ee483461c09), `Send` now returns `(Response, error)`


<a name="v1.0.3"></a>
# v1.0.3 (2014-11-25)

## :house: Housekeeping

- **concurrency**:
  - Improve concurrency support for Read operation ([380de829](https://github.com/StreamBoat/kodi_jsonrpc/commit/380de829d0eeadcaf5d457daa80d79c0404a3c6c))  
    <br>Also, add timeout support to initial connection


<a name="v1.0.2"></a>
# v1.0.2 (2014-11-25)

## :bug: Bug Fixes

- **reconnect**:
  - Add reconnect locking and actually update `Connected` ([32b5b63e](https://github.com/StreamBoat/kodi_jsonrpc/commit/32b5b63e3840c122abdc787e34e9f7c6ace16702))


<a name="v1.0.1"></a>
# v1.0.1 (2014-11-23)

## :bug: Bug Fixes

- **reconnect**:
  - Less painful reconnect ([a0f91e9d](https://github.com/StreamBoat/kodi_jsonrpc/commit/a0f91e9d89b2536700fe17d4c5709849a844203c))  <br>Vastly reduce the number of things touched by a reconnect, we only need
    to mess with the connection itself, and the attached encoders/decoders


<a name="v1.0.0"></a>
# v1.0.0 (2014-11-23)

## :house: Housekeeping

- **core**:
  - Rename from XBMC to Kodi, best to get this out of the way ([6e307398](https://github.com/StreamBoat/kodi_jsonrpc/commit/6e30739875014414562eb6ae11e7a30bc85e792c))
- **build**:
  - Add config for abe33/changelog-gen ([1502885c](https://github.com/StreamBoat/kodi_jsonrpc/commit/1502885c4d32f38850fc07b15215c0d29e0c23a2))  
    <br>TODO: Add contribution guidelines for commit messages
- **logging**:
  - Replace go-logging with logrus for structured logs ([0c4273a0](https://github.com/StreamBoat/kodi_jsonrpc/commit/0c4273a01011b2ca871ab7dfea61e7f8b123565e))

## Breaking Changes

- due to [6e307398](https://github.com/StreamBoat/kodi_jsonrpc/commit/6e30739875014414562eb6ae11e7a30bc85e792c), renaming the package represents a significant break for all existing users, which is why I wanted to get it done now, rather than at some future time.
- due to [0c4273a0](https://github.com/StreamBoat/kodi_jsonrpc/commit/0c4273a01011b2ca871ab7dfea61e7f8b123565e), `SetLogLevel` now takes a constant - one of: `LogDebugLevel`, `LogInfoLevel`, `LogWarnLevel`, `LogErrorLevel`, `LogFatalLevel`, `LogPanicLevel`

