import type { Nail, LineSegment, StringArtSettings } from "./string-art-engine"

const GO_BACKEND_URL = process.env.NEXT_PUBLIC_GO_BACKEND_URL || "http://localhost:9000"

export interface GoBackendProgress {
  line: LineSegment
  index: number
  total: number
}

export interface GoBackendResponse {
  lines: LineSegment[]
  nails: Nail[]
}

export async function* generateStringArtWithGoBackend(
  imageData: ImageData,
  nails: Nail[],
  settings: StringArtSettings
): AsyncGenerator<LineSegment, GoBackendResponse, undefined> {
  const width = imageData.width
  const height = imageData.height

  // Prepare request
  const request = {
    imageData: Array.from(imageData.data),
    width,
    height,
    nails,
    nailCount: settings.nailCount,
    lineCount: settings.lineCount,
    lineOpacity: settings.lineOpacity,
    shape: settings.shape,
    brightness: settings.brightness,
    contrast: settings.contrast,
    invertBrightness: settings.invertBrightness,
  }

  // Send request to Go backend
  const response = await fetch(`${GO_BACKEND_URL}/api/generate`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  })

  if (!response.ok) {
    throw new Error(`Backend error: ${response.status} ${response.statusText}`)
  }

  if (!response.body) {
    throw new Error("Response body is null")
  }

  // Read SSE stream
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ""

  let finalResponse: GoBackendResponse | null = null

  while (true) {
    const { done, value } = await reader.read()
    
    if (done) {
      // Process remaining buffer before breaking
      if (buffer.trim()) {
        const lines = buffer.split("\n")
        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const data = line.slice(6)
            if (data.trim()) {
              try {
                const parsed = JSON.parse(data)
                if (parsed.line && parsed.index !== undefined) {
                  yield parsed.line
                } else if (parsed.lines && parsed.nails) {
                  finalResponse = parsed
                }
              } catch (e) {
                console.error("Failed to parse final SSE data:", e, data)
              }
            }
          }
        }
      }
      break
    }

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split("\n")
    buffer = lines.pop() || ""

    for (const line of lines) {
      if (line.startsWith("data: ")) {
        const data = line.slice(6)
        if (data.trim()) {
          try {
            const parsed = JSON.parse(data)
            
            // Check if it's a progress update or final response
            if (parsed.line && parsed.index !== undefined) {
              // Progress update - yield the line
              yield parsed.line
            } else if (parsed.lines && parsed.nails) {
              // Final response - store it but continue processing
              finalResponse = parsed
              // Also yield all remaining lines from final response
              if (parsed.lines && parsed.lines.length > 0) {
                // Find which lines we haven't yielded yet
                // This ensures all lines are drawn
                for (const line of parsed.lines) {
                  yield line
                }
              }
            }
          } catch (e) {
            console.error("Failed to parse SSE data:", e, data)
          }
        }
      }
    }
  }

  if (!finalResponse) {
    throw new Error("No final response received from backend")
  }

  return finalResponse
}

export async function checkGoBackendHealth(): Promise<boolean> {
  try {
    const response = await fetch(`${GO_BACKEND_URL}/api/health`, {
      method: "GET",
    })
    return response.ok
  } catch {
    return false
  }
}

