/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Node } from "@xyflow/react"

interface Point {
  x: number
  y: number
}

interface Rect {
  left: number
  right: number
  top: number
  bottom: number
}

interface GridNode {
  i: number
  j: number
  x: number
  y: number
  g: number
  h: number
  f: number
  dir: 'H' | 'V' | null
  parent: GridNode | null
}

// Hanan Grid-based Obstacle Avoiding Router (libavoid-inspired, horizontally constrained)
export const getSmartPathPoints = (
  sourceX: number,
  sourceY: number,
  targetX: number,
  targetY: number,
  nodes: Node[],
  sourceId?: string,
  targetId?: string
): Point[] => {
  const margin = 28 // Compact safety margin around nodes (1 grid unit / 14px)

  // Check if source or target node is a waypoint to remove exit/entry stub margins
  const sourceNode = nodes.find((n) => n.id === sourceId)
  const targetNode = nodes.find((n) => n.id === targetId)
  const isSourceWaypoint = sourceNode?.type === "waypoint"
  const isTargetWaypoint = targetNode?.type === "waypoint"

  const exitBuffer = isSourceWaypoint ? 0 : 28
  const entryBuffer = isTargetWaypoint ? 0 : 28

  // Retrieve custom orientations (e.g. for Noop nodes). Fallback to standard Left-to-Right ('LR')
  const sourceOrientation = (sourceNode?.data?.orientation as 'LR' | 'RL' | 'TB' | 'BT') || 'LR'
  const targetOrientation = (targetNode?.data?.orientation as 'LR' | 'RL' | 'TB' | 'BT') || 'LR'

  // 1. Calculate exit and entry stub coordinates based on orientation directions
  let exitPt = { x: sourceX + exitBuffer, y: sourceY } // LR default
  if (sourceOrientation === 'RL') exitPt = { x: sourceX - exitBuffer, y: sourceY }
  else if (sourceOrientation === 'TB') exitPt = { x: sourceX, y: sourceY + exitBuffer }
  else if (sourceOrientation === 'BT') exitPt = { x: sourceX, y: sourceY - exitBuffer }

  let entryPt = { x: targetX - entryBuffer, y: targetY } // LR default
  if (targetOrientation === 'RL') entryPt = { x: targetX + entryBuffer, y: targetY }
  else if (targetOrientation === 'TB') entryPt = { x: targetX, y: targetY - entryBuffer }
  else if (targetOrientation === 'BT') entryPt = { x: targetX, y: targetY + entryBuffer }

  // 1. Extract obstacle rects from active nodes (including source and target to prevent self-crossing)
  const obstacles: Rect[] = nodes
    .filter((n) => n.type !== "waypoint")
    .map((n) => {
      const x = n.position.x
      const y = n.position.y
      const w = n.measured?.width || 112
      const h = n.measured?.height || 112

      return {
        left: x - margin,
        right: x + w + margin,
        top: y - margin,
        bottom: y + h + margin,
      }
    })

  // Helper to check if a horizontal segment intersects any obstacle
  const isHorizontalBlocked = (x1: number, x2: number, y: number): boolean => {
    const minX = Math.min(x1, x2)
    const maxX = Math.max(x1, x2)
    return obstacles.some(
      (rect) => y > rect.top && y < rect.bottom && minX < rect.right && maxX > rect.left
    )
  }

  // Helper to check if a vertical segment intersects any obstacle
  const isVerticalBlocked = (x: number, y1: number, y2: number): boolean => {
    const minY = Math.min(y1, y2)
    const maxY = Math.max(y1, y2)
    return obstacles.some(
      (rect) => x > rect.left && x < rect.right && minY < rect.bottom && maxY > rect.top
    )
  }

  // 2. Generate Hanan Grid lines (X and Y coordinates) using exitPt and entryPt
  const xs = [exitPt.x, entryPt.x]
  const ys = [exitPt.y, entryPt.y]

  obstacles.forEach((rect) => {
    xs.push(rect.left, rect.right)
    ys.push(rect.top, rect.bottom)
  })

  // Sort and deduplicate grid coordinates
  const uniqueXs = Array.from(new Set(xs)).sort((a, b) => a - b)
  const uniqueYs = Array.from(new Set(ys)).sort((a, b) => a - b)

  // Find start and end indices on the grid (using exitPt and entryPt)
  const startI = uniqueXs.indexOf(exitPt.x)
  const startJ = uniqueYs.indexOf(exitPt.y)
  const endI = uniqueXs.indexOf(entryPt.x)
  const endJ = uniqueYs.indexOf(entryPt.y)

  if (startI === -1 || startJ === -1 || endI === -1 || endJ === -1) {
    // Fallback if coordinates aren't found
    return getFallbackPath(
      sourceX,
      sourceY,
      targetX,
      targetY,
      sourceOrientation,
      targetOrientation,
      isSourceWaypoint,
      isTargetWaypoint
    )
  }

  // 3. A* Search on Hanan Grid
  const openList: GridNode[] = []
  const closedSet = new Set<string>()

  const toKey = (i: number, j: number) => `${i},${j}`
  const heuristic = (x1: number, y1: number, x2: number, y2: number) =>
    Math.abs(x1 - x2) + Math.abs(y1 - y2)

  const startNode: GridNode = {
    i: startI,
    j: startJ,
    x: exitPt.x,
    y: exitPt.y,
    g: 0,
    h: heuristic(exitPt.x, exitPt.y, entryPt.x, entryPt.y),
    f: heuristic(exitPt.x, exitPt.y, entryPt.x, entryPt.y),
    dir: 'H', // The stub arrived at this point horizontally
    parent: null,
  }

  openList.push(startNode)
  let destinationNode: GridNode | null = null

  const MAX_ITERATIONS = 500
  let iterations = 0

  while (openList.length > 0 && iterations < MAX_ITERATIONS) {
    iterations++

    // Sort open list by f score
    openList.sort((a, b) => a.f - b.f)
    const current = openList.shift()!

    const currentKey = toKey(current.i, current.j)
    closedSet.add(currentKey)

    // Check if reached destination
    if (current.i === endI && current.j === endJ) {
      destinationNode = current
      break
    }

    // Neighbors: Left, Right, Up, Down
    const neighbors: { i: number; j: number; dir: 'H' | 'V' }[] = []

    if (current.i > 0) neighbors.push({ i: current.i - 1, j: current.j, dir: 'H' })
    if (current.i < uniqueXs.length - 1) neighbors.push({ i: current.i + 1, j: current.j, dir: 'H' })
    if (current.j > 0) neighbors.push({ i: current.i, j: current.j - 1, dir: 'V' })
    if (current.j < uniqueYs.length - 1) neighbors.push({ i: current.i, j: current.j + 1, dir: 'V' })

    for (const neighbor of neighbors) {
      const nKey = toKey(neighbor.i, neighbor.j)
      if (closedSet.has(nKey)) continue

      const nX = uniqueXs[neighbor.i]
      const nY = uniqueYs[neighbor.j]

      // Check if segment is blocked by obstacles
      if (neighbor.dir === 'H') {
        if (isHorizontalBlocked(current.x, nX, current.y)) continue
      } else {
        if (isVerticalBlocked(current.x, current.y, nY)) continue
      }

      // Calculate edge cost
      const stepCost = neighbor.dir === 'H' ? Math.abs(nX - current.x) : Math.abs(nY - current.y)

      // Penalize turns to encourage straight routing segments
      const turnPenalty = (current.dir && current.dir !== neighbor.dir) ? 150 : 0
      const gScore = current.g + stepCost + turnPenalty
      const hScore = heuristic(nX, nY, entryPt.x, entryPt.y)
      const fScore = gScore + hScore

      const existingOpen = openList.find((n) => n.i === neighbor.i && n.j === neighbor.j)

      if (existingOpen) {
        if (gScore < existingOpen.g) {
          existingOpen.g = gScore
          existingOpen.f = fScore
          existingOpen.dir = neighbor.dir
          existingOpen.parent = current
        }
      } else {
        openList.push({
          i: neighbor.i,
          j: neighbor.j,
          x: nX,
          y: nY,
          g: gScore,
          h: hScore,
          f: fScore,
          dir: neighbor.dir,
          parent: current,
        })
      }
    }
  }

  // 4. Reconstruct path
  if (destinationNode) {
    const path: Point[] = []
    let curr: GridNode | null = destinationNode
    while (curr) {
      path.unshift({ x: curr.x, y: curr.y })
      curr = curr.parent
    }

    // Simplify collinear segments
    const innerPoints = simplifyPath(path)

    // Prepend start point and append end point to include strict horizontal stubs
    return [
      { x: sourceX, y: sourceY },
      ...innerPoints,
      { x: targetX, y: targetY }
    ]
  }

  return getFallbackPath(
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourceOrientation,
    targetOrientation,
    isSourceWaypoint,
    isTargetWaypoint
  )
}

// Simplifies consecutive horizontal/vertical collinear segments
const simplifyPath = (points: Point[]): Point[] => {
  if (points.length <= 2) return points

  const result: Point[] = [points[0]]

  for (let i = 1; i < points.length - 1; i++) {
    const prev = result[result.length - 1]
    const curr = points[i]
    const next = points[i + 1]

    const isCollinearX = Math.abs(prev.x - curr.x) < 0.1 && Math.abs(curr.x - next.x) < 0.1
    const isCollinearY = Math.abs(prev.y - curr.y) < 0.1 && Math.abs(curr.y - next.y) < 0.1

    if (isCollinearX || isCollinearY) {
      // Skip current point since it lies directly on the straight line between prev and next
      continue
    }
    result.push(curr)
  }

  result.push(points[points.length - 1])
  return result
}

// Fallback to simple step path with horizontal/vertical stubs if no obstacle-free path is found
const getFallbackPath = (
  sourceX: number,
  sourceY: number,
  targetX: number,
  targetY: number,
  sourceOrientation: 'LR' | 'RL' | 'TB' | 'BT' = 'LR',
  targetOrientation: 'LR' | 'RL' | 'TB' | 'BT' = 'LR',
  isSourceWaypoint = false,
  isTargetWaypoint = false
): Point[] => {
  const exitBuffer = isSourceWaypoint ? 0 : 28
  const entryBuffer = isTargetWaypoint ? 0 : 28

  let exitPt = { x: sourceX + exitBuffer, y: sourceY }
  if (sourceOrientation === 'RL') exitPt = { x: sourceX - exitBuffer, y: sourceY }
  else if (sourceOrientation === 'TB') exitPt = { x: sourceX, y: sourceY + exitBuffer }
  else if (sourceOrientation === 'BT') exitPt = { x: sourceX, y: sourceY - exitBuffer }

  let entryPt = { x: targetX - entryBuffer, y: targetY }
  if (targetOrientation === 'RL') entryPt = { x: targetX + entryBuffer, y: targetY }
  else if (targetOrientation === 'TB') entryPt = { x: targetX, y: targetY - entryBuffer }
  else if (targetOrientation === 'BT') entryPt = { x: targetX, y: targetY + entryBuffer }

  const isHorizExit = sourceOrientation === 'LR' || sourceOrientation === 'RL'
  const isHorizEntry = targetOrientation === 'LR' || targetOrientation === 'RL'

  if (isHorizExit && isHorizEntry) {
    const midX = exitPt.x + (entryPt.x - exitPt.x) / 2
    return [
      { x: sourceX, y: sourceY },
      exitPt,
      { x: midX, y: exitPt.y },
      { x: midX, y: entryPt.y },
      entryPt,
      { x: targetX, y: targetY }
    ]
  } else if (!isHorizExit && !isHorizEntry) {
    const midY = exitPt.y + (entryPt.y - exitPt.y) / 2
    return [
      { x: sourceX, y: sourceY },
      exitPt,
      { x: exitPt.x, y: midY },
      { x: entryPt.x, y: midY },
      entryPt,
      { x: targetX, y: targetY }
    ]
  } else {
    // Mixed: one vertical, one horizontal. Connect them directly at their intersection!
    const intersection = isHorizExit
      ? { x: entryPt.x, y: exitPt.y }
      : { x: exitPt.x, y: entryPt.y }
    return [
      { x: sourceX, y: sourceY },
      exitPt,
      intersection,
      entryPt,
      { x: targetX, y: targetY }
    ]
  }
}

export const getSmartPath = (
  sourceX: number,
  sourceY: number,
  targetX: number,
  targetY: number,
  nodes: Node[],
  sourceId?: string,
  targetId?: string
): string => {
  const points = getSmartPathPoints(sourceX, sourceY, targetX, targetY, nodes, sourceId, targetId)
  return pointsToSvgPath(points, 12)
}

// Converts array of points to SVG path with rounded corners (border radius)
const pointsToSvgPath = (points: Point[], borderRadius: number): string => {
  if (points.length === 0) return ""
  if (points.length === 1) return `M ${points[0].x} ${points[0].y}`

  let path = `M ${points[0].x} ${points[0].y}`
  let curX = points[0].x
  let curY = points[0].y

  for (let i = 1; i < points.length - 1; i++) {
    const prev = points[i - 1]
    const curr = points[i]
    const next = points[i + 1]

    const d1 = { x: curr.x - prev.x, y: curr.y - prev.y }
    const d2 = { x: next.x - curr.x, y: next.y - curr.y }

    const len1 = Math.sqrt(d1.x * d1.x + d1.y * d1.y)
    const len2 = Math.sqrt(d2.x * d2.x + d2.y * d2.y)

    const isCollinear = (Math.abs(d1.x) < 0.1 && Math.abs(d2.x) < 0.1) ||
      (Math.abs(d1.y) < 0.1 && Math.abs(d2.y) < 0.1)
    
    const r = isCollinear ? 0 : Math.min(borderRadius, len1 / 2, len2 / 2)

    if (r > 0) {
      const arcStart = {
        x: curr.x - (d1.x / len1) * r,
        y: curr.y - (d1.y / len1) * r,
      }
      const arcEnd = {
        x: curr.x + (d2.x / len2) * r,
        y: curr.y + (d2.y / len2) * r,
      }

      path += ` L ${arcStart.x} ${arcStart.y}`
      path += ` Q ${curr.x} ${curr.y}, ${arcEnd.x} ${arcEnd.y}`
      
      curX = arcEnd.x
      curY = arcEnd.y
    } else {
      path += ` L ${curr.x} ${curr.y}`
      curX = curr.x
      curY = curr.y
    }
  }

  const last = points[points.length - 1]
  path += ` L ${last.x} ${last.y}`

  return path
}
