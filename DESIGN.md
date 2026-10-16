---
name: Leafmark
colors:
  background: "#f5f7fb"
  surface: "#ffffff"
  ink: "#202d45"
  muted: "#61708a"
  primary: "#365bc4"
  border: "#dce2ed"
typography:
  body:
    fontFamily: "system-ui, sans-serif"
  editor:
    fontFamily: "Charter, 'Bitstream Charter', Georgia, serif"
spacing:
  unit: "4px"
rounded:
  control: "8px"
  panel: "16px"
---

# Leafmark design

## North star
A personal notebook on a blue drafting desk: orderly navigation, a generous writing sheet, and a folded-page mark. The writing is the memorable element. Product register throughout; no marketing landing page between returning users and their notes.

## Palette and typography
`static/app.css` owns runtime tokens. Its six named palette variables map directly to the frontmatter roles. System sans handles navigation and forms without external font requests; Charter/Georgia gives note writing a distinct reading rhythm. Muted copy stays legible. The accent belongs to primary actions and selected navigation.

## Layout
Desktop: 220px notebook rail, 320px searchable note list, remaining width writing sheet. Mobile: rail wraps above the list; opening a note replaces the list with an editor and a visible Back to notes action. Each pane owns its desktop scroll; mobile uses natural document scrolling.

## Shape and motion
Controls use 8px corners, major surfaces 16px. Borders communicate grouping; no ornamental gradients. Interaction feedback is immediate. Reduced motion disables transitions. Leafmark's folded-page symbol is shared by the favicon, app, and documentation.

## Content and states
Plain English, sentence case, explicit Save note. Never imply unsaved text reached the server. Save failure preserves the editor. No external assets, tracking, rendered HTML, or third-party scripts.
