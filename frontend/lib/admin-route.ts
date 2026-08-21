import { protectPanelRoute } from './session-route'

export const getServerSideProps = protectPanelRoute('admin')
