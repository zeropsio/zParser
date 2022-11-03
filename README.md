# YPARS

## Description

YAML parser with support for functions and string modifiers.

### Modifiers

Modifiers work on function calls and on static strings.
<details>
<summary>Example</summary>

Input

```yaml
  RAND_UPPER_STRING: "{$generateRandomString(50) | upper}"
  STATIC_UPPER_STRING: "{Static string that will be turned into upper case | upper}"
```

Output

```yaml
  RAND_UPPER_STRING: "oWkFjZJiu74QrhRmXb8GERGCkNYudqKT6ZUPYUFispstycrryo"
  STATIC_UPPER_STRING: "OWKFJZJIU74QRHRMXB8GERGCKNYUDQKT6ZUPYUFISPSTYCRRYO"
```

</details>

### Nesting

Function calls and strings with or without modifier may be nested even multiple layers deep.
<details>
<summary>Example</summary>

Input

```yaml
  NESTED_FUNCTIONS: "{$generateRandomString({$generateRandomInt(10, 50)})}"
  NESTED_STRINGS: "{My normal { And My NeStED | upper } strings}"
  NESTED_STRING_MODIFIERS: "{My normal { And My NeStED | upper } strings | sha256}"
  NESTED_STRING_IN_FUNCTION: "{$namedString(name, normal string {THAT IS ALL LOWER CASE | lower} even though middle part was not)}"
  YES_THIS_IS_EXCESSIVE: "{$namedString(randomLengthString, {$generateRandomString({$generateRandomInt({$generateRandomInt(10, 50)}, {$generateRandomInt(51, 100)})})}) | upper}"
```

Output

```yaml
  RAND_UPPER_STRING: "Lzkg38r27rq8xY6Y6VHvbqJ"
  NESTED_STRINGS: "My normal AND MY NESTED strings"
  NESTED_STRING_MODIFIERS: "57193af2e03136a21a6a88aace7a5ec5260689665a46b4eca5074656ac67a949"
  NESTED_STRING_IN_FUNCTION: "normal string that is all lower case even though middle part was not"
  YES_THIS_IS_EXCESSIVE: "FRQKTECZCSUEOCG6AOWJIRPJ2IWD623QZYUNSIJVEQEQH8APCMKJDGHU4MIZKTFGGEED6MGXJADA"
```

</details>

### Escaping

Characters can be escaped using backslash `\`. This also means it is mandatory to escape `\` like so `\\` for it to be
printed out.

One caveat is usage in languages like YAML. If used inside YAML's quoted strings, all backslashes `\` must be escaped
twice (`\\\\` instead of `\\`).  
This is needed so backslash is preserved through both YPARS and YAML parsing (as both remove one `\`).

<details>
<summary>Example</summary>

Input

```yaml
  ESCAPED_NESTED_FUNCTIONS: "\{$generateRandomString(\{$generateRandomInt(10, 50)\})\}"
  ESCAPE_TEST: "\{ \{\\\\ \\\\\\\\ \\\\\{ {\\\\} \\\\\} \\\\\\\\ \\\\\} \}"
  ESCAPE_TEST_WITH_ITEM: "\\\\{ sTriNG \\\\ witH, mOdiFiers | title }\\\\"
```

Output

```yaml
  ESCAPED_NESTED_FUNCTIONS: "{$generateRandomString({$generateRandomInt(10, 50)})}"
  ESCAPE_TEST: "{ {\\ \\\\ \\{ \\ \\} \\\\ \\} }"
  ESCAPE_TEST_WITH_ITEM: "\\STriNG \\ WitH, MOdiFiers\\"
```

Double escaping example

```yaml
    # Here is how following string will look like
    - "\{ \{\\\\ \\\\\\\\ \\\\\{ {\\\\} \\\\\} \\\\\\\\ \\\\\} \}"
    # - after our parsing
    - "{ {\\ \\\\ \\{ \\ \\} \\\\ \\} }"
    # - after additional yaml parsing
    - "{ {\ \\ \{ \ \} \\ \} }"
```

</details>

## Usage

### As a package

```shell
go get git.vsh-labs.cz/zerops/yaml-parser/src/parser
```

```go
package main

import (
	"log"
	"os"

	"git.vsh-labs.cz/zerops/yaml-parser/src/parser"
)

func main() {
	yml, err := os.Open("file.yml")
	if err != nil {
		log.Fatal(err)
	}

	p := parser.NewParser(yml, os.Stdout, 200)
	if err := p.Parse(); err != nil {
		log.Fatal(err)
	}
}
```

### As a binary

```shell
# output to stdOut
./bin/yamlParser-linux-amd64 ./example.yml

# output to file
./bin/yamlParser-linux-amd64 ./example.yml -f ./example.parsed.yml
```

## Supported functions

| name                      | description                                                                      | example                                     |
|---------------------------|----------------------------------------------------------------------------------|---------------------------------------------|
| generateRandomString      | generates random string in requested length                                      | `{$generateRandomString(50)}`               |
| generateRandomInt         | generates random in int range [min, max]                                         | `{$generateRandomInt(-999, 999)}`           |
| generateRandomNamedString | generates random string and stores it for later use                              | `{$generateRandomNamedString(myName, 50)}`  |
| namedString               | stores provided content for later use                                            | `{$namedString(myName, my string content)}` |
| getNamedString            | returns content of a stored string                                               | `{$getNamedString(myName)}`                 |
| getDateTime               | returns current date and time in specified format                                | `{$getDatetime(DD.MM.YYYY HH:mm:ss)}`       |
| generateED25519Key        | generates Public and Private ED25519 key pairs and stores them for later use     | `{$generateED25519Key(myEd25519Key)}`       |
| generateRSA4096Key        | generates Public and Private RSA 4096bit key pairs and stores them for later use | `{$generateRSA4096Key(myRSA4096Key)}`       |
| mercuryInRetrograde       | returns first parameter if Mercury IS in retrograde or second if it is not       | `{$mercuryInRetrograde(Yes, No)}`           |

---

### generateRandomString(length)

Generates random string in requested length
<details>

#### Parameters

| name   | type  | description                                        |
|--------|-------|----------------------------------------------------|
| length | `int` | required string length (max. allowed value `1024`) |

#### Example

| input                         | output               |
|-------------------------------|----------------------|
| `{$generateRandomString(20)}` | bc84df942e8290438c21 |
| `{$generateRandomString(10)}` | 94a2f484de           |

</details>

### `generateRandomInt(min, max)`

Generates random `int` in range `[min, max]`
<details>

#### Parameters

| name | type  | description                |
|------|-------|----------------------------|
| min  | `int` | minimum number (inclusive) |
| max  | `int` | maximum number (inclusive) |

#### Example

| input                             | output |
|-----------------------------------|--------|
| `{$generateRandomInt(-999, 999)}` | -155   |
| `{$generateRandomInt(0, 999999)}` | 6659   |

</details>

### `generateRandomNamedString(name, length)`

Generates random string and stores it for later use (using `getNamedString`) under provided name.
String is also returned as the output of the function call.

Any content that already existed under provided name is overwritten.
<details>

#### Parameters

| name   | type     | description                                                           |
|--------|----------|-----------------------------------------------------------------------|
| name   | `string` | name under which string may be retrieved later using `getNamedString` |
| length | `int`    | required string length (max. allowed value `1024`)                    |

#### Example

| input                                              | output                         |
|----------------------------------------------------|--------------------------------|
| `{$generateRandomNamedString(my20CharString, 30)}` | pj72x83UBgccTYfZRj3ytbApYeivq2 |
| `{$generateRandomNamedString(my15CharString, 15)}` | yhaKq7gyPoiVhwL                |
| `{$getNamedString(my15CharString)}`                | yhaKq7gyPoiVhwL                |

</details>

### `namedString(name, content)`

Stores provided content for later use under provided name.
String is also returned as the output of the function call.

Any content that already existed under provided name is overwritten.
<details>

#### Parameters

| name    | type     | description                                                           |
|---------|----------|-----------------------------------------------------------------------|
| name    | `string` | name under which string may be retrieved later using `getNamedString` |
| content | `string` | content to be stored                                                  |

#### Example

| input                                                                 | output                              |
|-----------------------------------------------------------------------|-------------------------------------|
| `{$namedString(myFirstString, content of my first string)}`           | content of my first string          |
| `{$namedString(mySecondString, content of my second string)}`         | content of my second string         |
| `{$namedString(mySecondString, updated content of my second string)}` | updated content of my second string |
| `{$getNamedString(mySecondString)}`                                   | updated content of my second string |

</details>

### `getNamedString(name)`

Returns content stored under provided name.
If no content was stored under provided name, error is returned.
<details>

#### Parameters

| name | type     | description                            |
|------|----------|----------------------------------------|
| name | `string` | name under which the content is stored |

#### Example

| input                                       | output                               |
|---------------------------------------------|--------------------------------------|
| `{$getNamedString(myExistingCustomString)}` | content of my existing custom string |
| `{$getNamedString(nonExistingString)}`      | `parsing will fail with an error`    |

</details>

### `getDateTime(format)`

Returns current date and time in specified format.
<details>

#### Parameters

| name   | type     | description                |
|--------|----------|----------------------------|
| format | `string` | see supported tokens below |

#### Supported tokens

| Type         | Token | Output                                  |
|--------------|-------|-----------------------------------------|
| Year         | YYYY  | 2000, 2001, 2002 … 2012, 2013           |
|              | YY    | 00, 01, 02 … 12, 13                     |
| Month        | MMMM  | January, February, March …              |
|              | MMM   | Jan, Feb, Mar …                         |
|              | MM    | 01, 02, 03 … 11, 12                     |
|              | M     | 1, 2, 3 … 11, 12                        |
| Day of Year  | DDDD  | 001, 002, 003 … 364, 365                |
| Day of Month | DD    | 01, 02, 03 … 30, 31                     |
|              | D     | 1, 2, 3 … 30, 31                        |
| Day of Week  | dddd  | Monday, Tuesday, Wednesday …            |
|              | ddd   | Mon, Tue, Wed …                         |
| Hour         | HH    | 00, 01, 02 … 23, 24                     |
|              | hh    | 01, 02, 03 … 11, 12                     |
|              | h     | 1, 2, 3 … 11, 12                        |
| AM / PM      | A     | AM, PM                                  |
|              | a     | am, pm                                  |
| Minute       | mm    | 00, 01, 02 … 58, 59                     |
|              | m     | 0, 1, 2 … 58, 59                        |
| Second       | ss    | 00, 01, 02 … 58, 59                     |
|              | s     | 0, 1, 2 … 58, 59                        |
| Microsecond  | S     | 000000 … 999999                         |
| Timezone     | ZZZ   | Asia/Baku, Europe/Warsaw, GMT           |
|              | zz    | -07:00, -06:00 … +06:00, +07:00, +08, Z |
|              | Z     | -0700, -0600 … +0600, +0700, +08, Z     |

#### Example

| input                                 | output              |
|---------------------------------------|---------------------|
| `{$getDatetime(DD.MM.YYYY HH:mm:ss)}` | 03.11.2022 12:32:35 |
| `{$getDatetime(DD.MM.YYYY)}`          | 03.11.2022          |

</details>

### generateED25519Key(name)

Generates Public and Private ED25519 key pairs and stores them for later use under name+suffix.

Function produces strings with newline characters and MUST be used with Literal scalar style.
Same goes for retrieval of stored key parts except public ssh key which is in one line. See example.
<details>

#### Parameters

| name | type     | description                                                             |
|------|----------|-------------------------------------------------------------------------|
| name | `string` | name under which all key versions will be stored (with suffixes bellow) |

#### Generated versions

| suffix     | description                                                     | example                                |
|------------|-----------------------------------------------------------------|----------------------------------------|
| Public     | public key version, also returned when the method is called     | `{$getNamedString(keyNamePublic)}`     |
| PublicSsh  | ssh formatted version used for authorized keys file             | `{$getNamedString(keyNamePublicSsh)}`  |
| Private    | private key version in standard format (not usable for OpenSSH) | `{$getNamedString(keyNamePrivate)}`    |
| PrivateSsh | private key version in Open SSH format                          | `{$getNamedString(keyNamePrivateSsh)}` |

#### Example

Input

```yaml
  MY_PUBLIC_KEY: |
    {$generateED25519Key(keyName)}

  MY_PUBLIC_SSH_KEY: "{$getNamedString(keyNamePublicSsh)}"

  MY_PRIVATE_KEY: |
    {$getNamedString(keyNamePrivate)}

  MY_PRIVATE_SSH_KEY: |
    {$getNamedString(keyNamePrivateSsh)}
```

Output

```yaml
  MY_PUBLIC_KEY: |
    -----BEGIN PUBLIC KEY-----
    MCowBQYDK2VwAyEA+1xKm4nA/ATJrQm9xX2fZj5PLxyApZURTDmDm5DQ4e0=
    -----END PUBLIC KEY-----

  MY_PUBLIC_SSH_KEY: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPtcSpuJwPwEya0JvcV9n2Y+Ty8cgKWVEUw5g5uQ0OHt"

  MY_PRIVATE_KEY: |
    -----BEGIN PRIVATE KEY-----
    MC4CAQAwBQYDK2VwBCIEIP9L2q781HpPRw0vgbiATskBeZNR4s5LXbGFKCm3V6iv
    -----END PRIVATE KEY-----

  MY_PRIVATE_SSH_KEY: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtz
    c2gtZWQyNTUxOQAAACD7XEqbicD8BMmtCb3FfZ9mPk8vHICllRFMOYObkNDh7QAA
    AIiaywRCmssEQgAAAAtzc2gtZWQyNTUxOQAAACD7XEqbicD8BMmtCb3FfZ9mPk8v
    HICllRFMOYObkNDh7QAAAED/S9qu/NR6T0cNL4G4gE7JAXmTUeLOS12xhSgpt1eo
    r/tcSpuJwPwEya0JvcV9n2Y+Ty8cgKWVEUw5g5uQ0OHtAAAAAAECAwQF
    -----END OPENSSH PRIVATE KEY-----
```

</details>

### generateRSA4096Key(name)

Generates Public and Private RSA 4096bit key pairs and stores them for later use under `name`+`suffix`.

Function produces strings with newline characters and MUST be used with Literal scalar style.
Same goes for retrieval of stored key parts except public ssh key which is in one line. See example.
<details>

#### Parameters

| name | type     | description                                                             |
|------|----------|-------------------------------------------------------------------------|
| name | `string` | name under which all key versions will be stored (with suffixes bellow) |

#### Generated versions

| suffix    | description                                                 | example                               |
|-----------|-------------------------------------------------------------|---------------------------------------|
| Public    | public key version, also returned when the method is called | `{$getNamedString(keyNamePublic)}`    |
| PublicSsh | ssh formatted version used for authorized keys file         | `{$getNamedString(keyNamePublicSsh)}` |
| Private   | private key version in standard format                      | `{$getNamedString(keyNamePrivate)}`   |

#### Example

Input

```yaml
  MY_PUBLIC_KEY: |
    {$generateED25519Key(keyName)}

  MY_PUBLIC_SSH_KEY: "{$getNamedString(keyNamePublicSsh)}"

  MY_PRIVATE_KEY: |
    {$getNamedString(keyNamePrivate)}
```

Output (keys are truncated)

```yaml
  MY_PUBLIC_KEY: |
    -----BEGIN PUBLIC KEY-----
    MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEArtYSRy47pzn4891nihMi
    yf21Pd6qEHuJATbBiFBNm/QHJU7Swt5WQGfyvOiSrV7TgwvjUv9nh3vasgV7uStL
    IbTJ3ZIQE/YOfNYWMZgEmxZjEcoH+6FRWRRIx6kXPiNfIynQr67F3afudrJ6ioyQ
    YpOwnyxeBKq7qHCTPUC030gWXQBzLGE+sFxzyXZO/FgQb5EidARL3pqMpKEVRRiv
    sm/PjVaYa6BvqQWlBXivTb7zBuAYxrZ24WY7Socx73UnaOzXVpXuF+rWROu7f73w
    5hIVpQi/CY5JnnElWqKJsSfddaeX0tERl+n/TC4Lutr9plITg3wWWLg+QJPlz9Nz
    IQXzZlc+CfHL1oJ4+hTfhc4zKbi30wg3H+WwMJFYrv1gwL5z9v/cMSzEOqc1cufL
    OEG7wN/7OKuXO9HbIvzMX3Vmx6N/CWk9NSSjedOYfytkvOBRnkOfU4Nc+PcC4XIc
    2+JfjokzFe1rpLmYNjz6Am9576xQPqdcS9rK2cLcuw1nC8oFZ6vRmz1CrLdWyV8H
    ZYqdMV3Qc6DMzDFw8PyI9uirTgIoo5j6sgIAz+DY0S7+ZpdDwUWxBJaDhOrVSt6d
    rh2jSPSDsmCa/keSU0mNKg/oj5ZBAWaDPM3PoS0Vr40O5KWGULSzzW+DiXDvqf7V
    YZ/vPEeiaWP4L9rXZBDMA78CAwEAAQ==
    -----END PUBLIC KEY-----

  MY_PUBLIC_SSH_KEY: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCu1hJHLjunOfjz3WeKEyLJ/bU93qoQe4kBNsGIUE2b9AclTtLC3lZAZ/....../gv2tdkEMwDvw=="

  MY_PRIVATE_KEY: |
    -----BEGIN PRIVATE KEY-----
    MIIJQwIBADANBgkqhkiG9w0BAQEFAASCCS0wggkpAgEAAoICAQCu1hJHLjunOfjz
    3WeKEyLJ/bU93qoQe4kBNsGIUE2b9AclTtLC3lZAZ/K86JKtXtODC+NS/2eHe9qy
    BXu5K0shtMndkhAT9g581hYxmASbFmMRygf7oVFZFEjHqRc+I18jKdCvrsXdp+52
    snqKjJBik7CfLF4EqruocJM9QLTfSBZdAHMsYT6wXHPJdk78WBBvkSJ0BEvemoyk
    oRVFGK+yb8+NVphroG+pBaUFeK9NvvMG4BjGtnbhZjtKhzHvdSdo7NdWle4X6tZE
    67t/vfDmEhWlCL8JjkmecSVaoomxJ911p5fS0RGX6f9MLgu62v2mUhODfBZYuD5A
    k+XP03MhBfNmVz4J8cvWgnj6FN+FzjMpuLfTCDcf5bAwkViu/WDAvnP2/9wxLMQ6
    pzVy58s4QbvA3/s4q5c70dsi/MxfdWbHo38JaT01JKN505h/K2S84FGeQ59Tg1z4
    9wLhchzb4l+OiTMV7WukuZg2PPoCb3nvrFA+p1xL2srZwty7DWcLygVnq9GbPUKs
    t1bJXwdlip0xXdBzoMzMMXDw/Ij26KtOAiijmPqyAgDP4NjRLv5ml0PBRbEEloOE
    ......
    qGNempfsElD4BbAtxwko8NqeTANjBiE=
    -----END PRIVATE KEY-----

```

</details>

###

<details>
<summary>🥚</summary>

### `mercuryInRetrograde(contentIfYes, contentIfNo)`

Returns first parameter if Mercury IS in retrograde or second if it is not.
<details>

#### Parameters

| name         | type     | description                                            |
|--------------|----------|--------------------------------------------------------|
| contentIfYes | `string` | content to be returned if Mercury IS in retrograde     |
| contentIfNo  | `string` | content to be returned if Mercury IS NOT in retrograde |

#### Example

| input                             | output |
|-----------------------------------|--------|
| `{$mercuryInRetrograde(Yes, No)}` | No     |

</details>
</details>

---

## Supported modifiers

| name     | description                                                             |
|----------|-------------------------------------------------------------------------|
| sha256   | hashes string using sha256 algorithm                                    |
| sha512   | hashes string using sha512 algorithm                                    |
| bcrypt   | hashes string using bcrypt algorithm                                    |
| argon2id | hashes string using argon2id algorithm                                  |
| upper    | maps all unicode letters to their upper case                            |
| lower    | maps all unicode letters to their lower case                            |
| title    | maps all words to title case (first letter upper case, rest lower case) |
| noop     | does nothing - used in tests                                            |

### Examples

| input                                                      | output                                                                                                                           |
|------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------|
| `{$generateRandomNamedString(myPassword, 30)}`             | 7a14c8e74bc98a0d74253b1d1a4ef6                                                                                                   |
| <code>{$getNamedString(myPassword) &#124; sha256}</code>   | 081b91d6dff5036229a92e2442fb65d7c8124571d4e70a2ac4729aeb86957407                                                                 |
| <code>{$getNamedString(myPassword) &#124; sha512}</code>   | 89c05547de0aa4926512a958f95ab8bf4096ceec63ad5aad4266890bfa059e0cc98917c54276ba4cd61f1dde4c8efda948fc967885c9dd50558ed939722ca10c |
| <code>{$getNamedString(myPassword) &#124; bcrypt}</code>   | $2a$10$CxKZX0yIxdc7ts6eI5aBu.g.heAsFcePdMDEpnlViTlo3vGc//PXe                                                                     |
| <code>{$getNamedString(myPassword) &#124; argon2id}</code> | $argon2id$v=19$m=98304,t=1,p=3$uWBpmoUT3sfckXHyRF9hlg$8bGtNffuHxaRIgN99zCmJeGEYJF5BY2J9TwzqmezP28                                |
| <code>{sTATic StrINg wiTH a mOdifIER &#124; upper}</code>  | STATIC STRING WITH A MODIFIER                                                                                                    |
| <code>{sTATic StrINg wiTH a mOdifIER &#124; lower}</code>  | static string with a modifier                                                                                                    |
| <code>{sTATic StrINg wiTH a mOdifIER &#124; title}</code>  | Static String With A Modifier                                                                                                    |
| <code>{sTATic StrINg wiTH a mOdifIER &#124; noop}</code>   | sTATic StrINg wiTH a mOdifIER                                                                                                    |

### Bcrypt configuration

- cost: `10`

### Argon2id configuration

- saltLen: `16B`
- memory: `96MiB`
- iterations: `1`
- parallelism: `3`
- keyLength: `32B`
