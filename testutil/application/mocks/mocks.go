package mocks

// This file is in place to declare the package for dynamically generated mocks
class Solution:
    # Do I have >= 3 papers w/ 3 citations
    # Do I have >= 4 papers w/ 4  citations
    # Do I have >= N papers w/ N  citations
    # Array[i] => # of papers w/ >= i citations
    def hIndex(self, citations: List[int]) -> int:

        # Find the largest number of references
        maxC = -1
        for c in citations:
            maxC = maxC if c < maxC else c

        # O(n^2) runtime + O(n) memory solution
        # # Determine the number of papers with i references or more
        # hList = [0] * (maxC + 1)
        # for c in citations:
        #     for i in range(0, c + 1):
        #         hList[i] += 1

        # # Find the hIndex
        # hIndex = 0
        # for num_citations in range(len(hList)):
        #     num_papers = hList[num_citations]
        #     if num_papers >= num_citations:
        #         hIndex = num_citations