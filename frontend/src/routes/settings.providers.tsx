import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/settings/providers')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/settings/providers"!</div>
}
