import { mount } from '@vue/test-utils'
import { expect } from 'chai'
import { createVuetify } from 'vuetify'
import TreeItem from '@/components/TreeItem.vue'

describe('TreeItem.vue', () => {
  let vuetify

  beforeEach(() => {
    vuetify = createVuetify()
  })

  it('renders', () => {
    const wrapper = mount(TreeItem, {
      global: {
        plugins: [vuetify],
      },
      props: {
        item: { isRoot: true },
      },
    })
    expect(wrapper.html()).to.include('item')
  })
})
