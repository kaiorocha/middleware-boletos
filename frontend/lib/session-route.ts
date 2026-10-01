import type { GetServerSidePropsContext } from 'next'

const COOKIE_NAME = 'giga_panel_session'

function cookieRole(context: GetServerSidePropsContext) {
  const cookies = context.req.headers.cookie || ''
  const value = cookies.split(';').map((part) => part.trim()).find((part) => part.startsWith(`${COOKIE_NAME}=`))
  return value?.slice(COOKIE_NAME.length + 1)
}

export function protectPanelRoute(expectedRole: 'admin' | 'tenant') {
  return async function getServerSideProps(context: GetServerSidePropsContext) {
    const role = cookieRole(context)
    if (!role) return { redirect: { destination: '/login', permanent: false } }
    if (role !== expectedRole) return { redirect: { destination: role === 'admin' ? '/admin' : '/app', permanent: false } }
    return { props: {} }
  }
}

export async function redirectAuthenticatedUser(context: GetServerSidePropsContext) {
  const role = cookieRole(context)
  if (role === 'admin' || role === 'tenant') {
    return { redirect: { destination: role === 'admin' ? '/admin' : '/app', permanent: false } }
  }
  return { props: {} }
}
