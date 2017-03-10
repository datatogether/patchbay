package main

import (
	"fmt"
	"time"
)

func (c *Client) ArchiveUrl(db sqlQueryExecable, reqId, url string) {
	logger.Println("archiving %s", url)
	u := &Url{Url: url}
	if _, err := u.ParsedUrl(); err != nil {
		logger.Println(err.Error())
		c.SendResponse(&ClientResponse{
			Type:      "URL_ARCHIVE_ERROR",
			RequestId: reqId,
			Error:     fmt.Sprintf("url parse error: %s", err.Error()),
		})
		return
	}

	if err := u.Read(db); err != nil {
		if err == ErrNotFound {
			if err := u.Insert(db); err != nil {
				logger.Println(err.Error())
				c.SendResponse(&ClientResponse{
					Type:      "URL_ARCHIVE_ERROR",
					RequestId: reqId,
					Error:     fmt.Sprintf("internal server error"),
				})
				return
			}
		} else {
			logger.Println(err.Error())
			c.SendResponse(&ClientResponse{
				Type:      "URL_ARCHIVE_ERROR",
				RequestId: reqId,
				Error:     fmt.Sprintf("internal server error"),
			})
			return
		}
	}

	// Initial get succeeded, let the client know
	c.SendResponse(&ClientResponse{
		Type:      "URL_ARCHIVE_SUCCESS",
		RequestId: reqId,
		Schema:    "URL",
		Data:      u,
	})

	// Perform base GET request
	links, err := u.Get(db, func(err error) {
		if err != nil {
			logger.Println(err.Error())
			c.SendResponse(&ClientResponse{
				Type:      "URL_ARCHIVE_ERROR",
				RequestId: reqId,
				Error:     fmt.Sprintf("error getting url: %s", err.Error()),
			})
			return
		}
	})
	if err != nil {
		logger.Println(err.Error())
		c.SendResponse(&ClientResponse{
			Type:      "URL_ARCHIVE_ERROR",
			RequestId: reqId,
			Error:     fmt.Sprintf("internal server error"),
		})
		return
	}

	// push our new links to client
	c.SendResponse(&ClientResponse{
		Type:      FetchOutboundLinksAct{}.SuccessType(),
		RequestId: "server",
		Schema:    "LINK_ARRAY",
		Data:      links,
	})

	go func(db sqlQueryExecable, links []*Link) {
		// GET each destination link from this page in parallel
		for _, l := range links {
			// need a sleep here to avoid bombing server with requests
			// tooooo hard, also we sleep first b/c the websocket trips up if
			// we jam the messages to hard.
			time.Sleep(time.Second * 3)

			c.SendResponse(&ClientResponse{
				Type:      "URL_SET_LOADING",
				RequestId: "server",
				Data: map[string]interface{}{
					"url":     l.Dst.Url,
					"loading": true,
				},
			})

			if _, err := l.Dst.Get(db, func(err error) {
				if err != nil {
					c.SendResponse(&ClientResponse{
						Type:      "URL_SET_ERROR",
						RequestId: "server",
						Data: map[string]interface{}{
							"url":   l.Dst.Url,
							"error": err.Error(),
						},
					})
				}
				c.SendResponse(&ClientResponse{
					Type:      "URL_SET_SUCCESS",
					RequestId: "server",
					Data: map[string]interface{}{
						"url":     l.Dst.Url,
						"success": true,
					},
				})
				// taskDone(err)
			}); err != nil {
				logger.Println(err.Error())
				c.SendResponse(&ClientResponse{
					Type:      "URL_SET_ERROR",
					RequestId: "server",
					Data: map[string]interface{}{
						"url":   l.Dst.Url,
						"error": err.Error(),
					},
				})
			}
		}
	}(db, links)
}

// ArchiveUrl GET's a url and if it's an HTML page, any links it directly references
func ArchiveUrl(db sqlQueryExecable, url string, done func(err error)) (*Url, []*Link, error) {
	u := &Url{Url: url}
	if _, err := u.ParsedUrl(); err != nil {
		done(err)
		return nil, nil, err
	}

	if err := u.Read(db); err != nil {
		if err == ErrNotFound {
			if err := u.Insert(db); err != nil {
				done(err)
				return nil, nil, err
			}
		} else {
			done(err)
			return nil, nil, err
		}
	}

	// Perform GET request
	links, err := u.Get(db, func(err error) {
		if err != nil {
			done(err)
		}
	})
	if err != nil {
		done(err)
		return u, links, err
	}

	tasks := len(links)
	errs := make(chan error, tasks)
	taskDone := func(err error) {
		errs <- err
	}

	go func(db sqlQueryExecable, links []*Link) {
		// GET each destination link from this page in parallel
		for _, l := range links {
			if _, err := l.Dst.Get(db, taskDone); err != nil {
				logger.Println(err.Error())
			}

			// need a sleep here to avoid bombing server with requests
			// tooooo hard
			time.Sleep(time.Second * 3)
		}
	}(db, links)

	go func() {
		for i := 0; i < tasks; i++ {
			err := <-errs
			if err != nil {
				done(err)
				return
			}
		}
		done(nil)
	}()

	return u, links, err
}

func ArchiveUrlSync(db sqlQueryExecable, url string) (*Url, error) {
	done := make(chan error)
	u, _, err := ArchiveUrl(db, url, func(err error) {
		done <- err
	})
	if err != nil {
		return u, err
	}

	err = <-done
	return u, err
}
