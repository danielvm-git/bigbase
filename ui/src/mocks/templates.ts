export interface MessageTemplate {
  id: string
  name: string
  type: 'email' | 'sms' | 'push'
  subject: string
  body: string
  variables: string[]
  status: 'draft' | 'active' | 'archived'
  updated_at: string
}

export const mockTemplates: MessageTemplate[] = [
  {
    id: 'tpl-welcome',
    name: 'Welcome email',
    type: 'email',
    subject: 'Welcome to {{app_name}}',
    body: 'Hi {{user_name}},\n\nThanks for joining {{app_name}}!',
    variables: ['app_name', 'user_name'],
    status: 'active',
    updated_at: '2026-05-12T10:00:00Z',
  },
  {
    id: 'tpl-reset',
    name: 'Password reset',
    type: 'email',
    subject: 'Reset your password',
    body: 'Click here to reset: {{reset_link}}',
    variables: ['reset_link'],
    status: 'active',
    updated_at: '2026-05-10T14:30:00Z',
  },
  {
    id: 'tpl-otp',
    name: 'SMS verification',
    type: 'sms',
    subject: '',
    body: 'Your code is {{code}}. Expires in 10 minutes.',
    variables: ['code'],
    status: 'draft',
    updated_at: '2026-05-08T09:15:00Z',
  },
  {
    id: 'tpl-push',
    name: 'New message alert',
    type: 'push',
    subject: 'New message',
    body: '{{sender}} sent you a message',
    variables: ['sender'],
    status: 'active',
    updated_at: '2026-05-01T16:00:00Z',
  },
]

export function getTemplate(id: string): MessageTemplate | undefined {
  return mockTemplates.find(t => t.id === id)
}
