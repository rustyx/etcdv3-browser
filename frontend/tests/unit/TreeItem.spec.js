import { mount } from '@vue/test-utils'
import { expect } from 'chai'
import TreeItem from '@/components/TreeItem.vue'

describe('TreeItem.vue', () => {
  it('renders', () => {
    const wrapper = mount(TreeItem, {
      global: {
        stubs: {
          'v-icon': true,
        },
      },
      props: {
        item: { isRoot: true },
      },
    })
    expect(wrapper.html()).to.include('item')
  })
})
