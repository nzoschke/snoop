package snoop

  origin := fmt.Sprintf("https://%s", c.Host)
  endpoint := fmt.Sprintf("wss://%s%s", c.Host, path)

  config, err := websocket.NewConfig(endpoint, origin)

  if err != nil {
    return err
  }
