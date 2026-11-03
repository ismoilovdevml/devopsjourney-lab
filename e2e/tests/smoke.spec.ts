import { test, expect } from '../fixtures'

test('student lands on the dashboard after registering', async ({ page, student }) => {
  await page.goto('/')
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByText(student.username).first()).toBeVisible()
})
