# YouTube API Integration Patterns

## YouTube Data API v3

Feedmix uses the YouTube Data API v3 to fetch subscription data and video information.

**Base URL**: `https://www.googleapis.com/youtube/v3`
**Documentation**: https://developers.google.com/youtube/v3

## Quota System

### Quota Costs
Each API request costs quota units:
- `subscriptions.list`: 1 unit per request (50 items max per request)
- `videos.list`: 1 unit per request (50 items max per request)
- `channels.list`: 1 unit per request (50 items max per request)

**Daily Quota**: 10,000 units/day (default for free tier)

### Quota Optimization
- Batch requests when possible (use `id` parameter with comma-separated list)
- Cache responses locally (subscriptions don't change frequently)
- Request only needed fields using `part` parameter

## API Endpoints

### Subscriptions

**Endpoint**: `GET /youtube/v3/subscriptions`

**Purpose**: List user's YouTube subscriptions (channels they follow)

**Request**:
```http
GET /youtube/v3/subscriptions?part=snippet&mine=true&maxResults=50
Authorization: Bearer ACCESS_TOKEN
```

**Response**:
```json
{
  "items": [
    {
      "snippet": {
        "title": "Channel Name",
        "description": "Channel description",
        "resourceId": {
          "channelId": "UCxxxxxxxxxxxxxxxxxxxx"
        },
        "thumbnails": {
          "default": { "url": "https://..." },
          "medium": { "url": "https://..." },
          "high": { "url": "https://..." }
        }
      }
    }
  ],
  "nextPageToken": "CAUQAA"
}
```

**Pagination**:
- `maxResults`: Max 50 items per request
- `pageToken`: Use `nextPageToken` from response for next page
- Loop until `nextPageToken` is absent

**Implementation**:
```go
func (c *Client) FetchSubscriptions() ([]Subscription, error) {
    var allSubs []Subscription
    pageToken := ""

    for {
        url := fmt.Sprintf(
            "https://www.googleapis.com/youtube/v3/subscriptions?part=snippet&mine=true&maxResults=50&pageToken=%s",
            pageToken,
        )

        req, _ := http.NewRequest("GET", url, nil)
        req.Header.Set("Authorization", "Bearer "+c.accessToken)

        resp, err := c.httpClient.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()

        var result SubscriptionsResponse
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            return nil, err
        }

        allSubs = append(allSubs, result.Items...)

        if result.NextPageToken == "" {
            break
        }
        pageToken = result.NextPageToken
    }

    return allSubs, nil
}
```

### Videos

**Endpoint**: `GET /youtube/v3/videos`

**Purpose**: Get video details (title, description, publish date, etc.)

**Request**:
```http
GET /youtube/v3/videos?part=snippet,contentDetails&id=VIDEO_ID1,VIDEO_ID2
Authorization: Bearer ACCESS_TOKEN
```

**Response**:
```json
{
  "items": [
    {
      "id": "dQw4w9WgXcQ",
      "snippet": {
        "title": "Video Title",
        "description": "Video description",
        "publishedAt": "2024-01-15T10:30:00Z",
        "channelId": "UCxxxxxxxxxxxxxxxxxxxx",
        "channelTitle": "Channel Name",
        "thumbnails": {
          "default": { "url": "https://..." }
        }
      },
      "contentDetails": {
        "duration": "PT4M33S"
      }
    }
  ]
}
```

**Batching**:
- Request up to 50 video IDs per request (comma-separated)
- Reduces quota usage: 1 request for 50 videos vs 50 separate requests

### Channels

**Endpoint**: `GET /youtube/v3/channels`

**Purpose**: Get channel details and latest uploads

**Request**:
```http
GET /youtube/v3/channels?part=snippet,contentDetails&id=CHANNEL_ID
Authorization: Bearer ACCESS_TOKEN
```

**Response**:
```json
{
  "items": [
    {
      "id": "UCxxxxxxxxxxxxxxxxxxxx",
      "snippet": {
        "title": "Channel Name",
        "description": "Channel description"
      },
      "contentDetails": {
        "relatedPlaylists": {
          "uploads": "UUxxxxxxxxxxxxxxxxxxxx"
        }
      }
    }
  ]
}
```

**Getting Latest Uploads**:
1. Get channel's uploads playlist ID from `contentDetails.relatedPlaylists.uploads`
2. Use `playlistItems.list` to get videos from uploads playlist

### Playlist Items

**Endpoint**: `GET /youtube/v3/playlistItems`

**Purpose**: List videos in a playlist (used for getting channel uploads)

**Request**:
```http
GET /youtube/v3/playlistItems?part=snippet,contentDetails&playlistId=UPLOADS_PLAYLIST_ID&maxResults=50
Authorization: Bearer ACCESS_TOKEN
```

**Response**:
```json
{
  "items": [
    {
      "snippet": {
        "publishedAt": "2024-01-15T10:30:00Z",
        "title": "Video Title",
        "resourceId": {
          "videoId": "dQw4w9WgXcQ"
        }
      }
    }
  ]
}
```

## Error Handling

### Common Error Codes

**401 Unauthorized**
- Cause: Invalid or expired access token
- Solution: Refresh token or re-authenticate

**403 Forbidden**
- Cause: Quota exceeded or API not enabled
- Solution: Wait for quota reset (midnight Pacific Time) or enable API in Google Cloud Console

**404 Not Found**
- Cause: Invalid video/channel ID
- Solution: Skip invalid IDs, log for debugging

**429 Too Many Requests**
- Cause: Rate limiting (100 requests per 100 seconds per user)
- Solution: Implement exponential backoff

### Error Handling Pattern

```go
func (c *Client) fetchWithRetry(url string, maxRetries int) (*http.Response, error) {
    var resp *http.Response
    var err error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        resp, err = c.fetch(url)
        if err != nil {
            return nil, err
        }

        switch resp.StatusCode {
        case 200:
            return resp, nil

        case 401:
            // Try to refresh token
            if err := c.refreshToken(); err != nil {
                return nil, ErrNotAuthenticated
            }
            continue // Retry with new token

        case 403:
            // Check if quota exceeded
            body, _ := io.ReadAll(resp.Body)
            if strings.Contains(string(body), "quotaExceeded") {
                return nil, ErrQuotaExceeded
            }
            return nil, ErrForbidden

        case 429:
            // Exponential backoff
            backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            time.Sleep(backoff)
            continue

        default:
            return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
        }
    }

    return nil, fmt.Errorf("max retries exceeded")
}
```

## Data Aggregation Strategy

### Efficient Feed Generation

**Goal**: Show recent videos from subscribed channels

**Strategy**:
1. Fetch all subscriptions (paginated, cache for 24 hours)
2. For each channel, get uploads playlist ID
3. Fetch recent videos from each uploads playlist (up to 10 per channel)
4. Aggregate all videos, sort by publish date (descending)
5. Display top N videos (default: 50)

**Optimization**:
- Cache subscription list (rarely changes)
- Batch video fetches (50 videos per request)
- Limit videos per channel (avoid spam channels dominating feed)
- Only fetch snippet (don't need statistics, contentDetails for feed)

### Implementation

```go
func (a *Aggregator) GenerateFeed(limit int) ([]Video, error) {
    // 1. Get subscriptions (cached)
    subs, err := a.client.FetchSubscriptions()
    if err != nil {
        return nil, err
    }

    // 2. Collect video IDs from all channels
    var videoIDs []string
    for _, sub := range subs {
        channelVideos, err := a.client.FetchChannelVideos(sub.ChannelID, 10)
        if err != nil {
            log.Printf("Error fetching videos for %s: %v", sub.Title, err)
            continue // Skip failed channels
        }

        for _, v := range channelVideos {
            videoIDs = append(videoIDs, v.ID)
        }
    }

    // 3. Batch fetch video details (50 per request)
    var allVideos []Video
    for i := 0; i < len(videoIDs); i += 50 {
        end := min(i+50, len(videoIDs))
        batch := videoIDs[i:end]

        videos, err := a.client.FetchVideos(batch)
        if err != nil {
            return nil, err
        }

        allVideos = append(allVideos, videos...)
    }

    // 4. Sort by publish date (newest first)
    sort.Slice(allVideos, func(i, j int) bool {
        return allVideos[i].PublishedAt.After(allVideos[j].PublishedAt)
    })

    // 5. Return top N
    if len(allVideos) > limit {
        allVideos = allVideos[:limit]
    }

    return allVideos, nil
}
```

## Testing

### Contract Tests

Verify API response structure hasn't changed:
```go
func TestYouTubeAPI_SubscriptionsEndpoint(t *testing.T) {
    client := setupRealClient(t)

    resp, err := client.FetchSubscriptions()
    require.NoError(t, err)

    // Verify structure
    assert.NotEmpty(t, resp.Items)
    item := resp.Items[0]
    assert.NotEmpty(t, item.Snippet.Title)
    assert.NotEmpty(t, item.Snippet.ResourceID.ChannelID)
}
```

### Unit Tests (Mock HTTP)

```go
func TestClient_FetchSubscriptions(t *testing.T) {
    mockHTTP := &MockHTTPClient{
        Response: &http.Response{
            StatusCode: 200,
            Body: io.NopCloser(strings.NewReader(`{
                "items": [
                    {
                        "snippet": {
                            "title": "Test Channel",
                            "resourceId": {"channelId": "UC123"}
                        }
                    }
                ]
            }`)),
        },
    }

    client := NewYouTubeClient(mockHTTP)
    subs, err := client.FetchSubscriptions()

    assert.NoError(t, err)
    assert.Len(t, subs, 1)
    assert.Equal(t, "Test Channel", subs[0].Title)
}
```

## Common Issues

### Issue: "The request cannot be completed because you have exceeded your quota"
**Solution**:
- Wait for quota reset (midnight Pacific Time)
- Implement caching to reduce API calls
- Request quota increase in Google Cloud Console

### Issue: Videos missing from feed
**Cause**: Uploads playlist fetch failed, channel has no videos, or channel deleted
**Solution**: Log errors, skip failed channels, continue with others

### Issue: Slow feed generation (>10 seconds)
**Cause**: Too many API requests, no batching
**Solution**: Batch video fetches (50 per request), cache subscriptions, limit videos per channel

## References
- [YouTube Data API v3 Documentation](https://developers.google.com/youtube/v3)
- [Quota Calculator](https://developers.google.com/youtube/v3/determine_quota_cost)
- [API Explorer](https://developers.google.com/youtube/v3/docs)
