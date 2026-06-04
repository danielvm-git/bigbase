/** Step-3 headline for the create-site deploy wizard. */
export function deployWizardTitle(deploying: boolean, status: string): string {
  if (deploying || status === 'pending' || status === 'building') {
    return 'Building your site…'
  }
  if (status === 'failed') {
    return 'Deploy failed'
  }
  return 'Your site is live'
}
