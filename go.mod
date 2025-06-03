module Smart-Music-Go

go 1.23.8

require (
    github.com/zmb3/spotify v0.0.0
    golang.org/x/oauth2 v0.0.0
)

replace github.com/zmb3/spotify => ./internal/stub/spotify
replace golang.org/x/oauth2 => ./internal/stub/oauth2
